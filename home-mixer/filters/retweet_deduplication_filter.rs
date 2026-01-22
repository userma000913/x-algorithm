use crate::candidate_pipeline::candidate::PostCandidate;
use crate::candidate_pipeline::query::ScoredPostsQuery;
use std::collections::HashSet;
use tonic::async_trait;
use xai_candidate_pipeline::filter::{Filter, FilterResult};

/// 对转推进行去重，只保留推文的首次出现
/// （无论是原始推文还是转推）。
pub struct RetweetDeduplicationFilter;

#[async_trait]
impl Filter<ScoredPostsQuery, PostCandidate> for RetweetDeduplicationFilter {
    async fn filter(
        &self,
        _query: &ScoredPostsQuery,
        candidates: Vec<PostCandidate>,
    ) -> Result<FilterResult<PostCandidate>, String> {
        let mut seen_tweet_ids: HashSet<u64> = HashSet::new();
        let mut kept = Vec::new();
        let mut removed = Vec::new();

        for candidate in candidates {
            match candidate.retweeted_tweet_id {
                Some(retweeted_id) => {
                    // 如果已经见过这条推文（作为原始推文或转推），则移除
                    if seen_tweet_ids.insert(retweeted_id) {
                        kept.push(candidate);
                    } else {
                        removed.push(candidate);
                    }
                }
                None => {
                    // 标记这条原始推文ID为已见，以便过滤掉它的转推
                    seen_tweet_ids.insert(candidate.tweet_id as u64);
                    kept.push(candidate);
                }
            }
        }

        Ok(FilterResult { kept, removed })
    }
}
