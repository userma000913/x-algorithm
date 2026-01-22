use crate::candidate_pipeline::candidate::PostCandidate;
use crate::candidate_pipeline::query::ScoredPostsQuery;
use tonic::async_trait;
use xai_candidate_pipeline::filter::{Filter, FilterResult};

/// 过滤掉作者是查看者本人的推文。
pub struct SelfTweetFilter;

#[async_trait]
impl Filter<ScoredPostsQuery, PostCandidate> for SelfTweetFilter {
    async fn filter(
        &self,
        query: &ScoredPostsQuery,
        candidates: Vec<PostCandidate>,
    ) -> Result<FilterResult<PostCandidate>, String> {
        let viewer_id = query.user_id as u64;
        let (kept, removed): (Vec<_>, Vec<_>) = candidates
            .into_iter()
            .partition(|c| c.author_id != viewer_id);

        Ok(FilterResult { kept, removed })
    }
}
