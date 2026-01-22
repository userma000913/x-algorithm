use lazy_static::lazy_static;
use log::{debug, info, warn};
use std::cmp::Reverse;
use std::collections::HashSet;
use std::sync::Arc;
use std::time::{Instant, SystemTime, UNIX_EPOCH};
use tokio::sync::Semaphore;
use tonic::{Request, Response, Status};

use xai_thunder_proto::{
    GetInNetworkPostsRequest, GetInNetworkPostsResponse, LightPost,
    in_network_posts_service_server::{InNetworkPostsService, InNetworkPostsServiceServer},
};

use crate::config::{
    MAX_INPUT_LIST_SIZE, MAX_POSTS_TO_RETURN, MAX_VIDEOS_TO_RETURN,
};
use crate::metrics::{
    GET_IN_NETWORK_POSTS_COUNT, GET_IN_NETWORK_POSTS_DURATION,
    GET_IN_NETWORK_POSTS_DURATION_WITHOUT_STRATO, GET_IN_NETWORK_POSTS_EXCLUDED_SIZE,
    GET_IN_NETWORK_POSTS_FOLLOWING_SIZE, GET_IN_NETWORK_POSTS_FOUND_FRESHNESS_SECONDS,
    GET_IN_NETWORK_POSTS_FOUND_POSTS_PER_AUTHOR, GET_IN_NETWORK_POSTS_FOUND_REPLY_RATIO,
    GET_IN_NETWORK_POSTS_FOUND_TIME_RANGE_SECONDS, GET_IN_NETWORK_POSTS_FOUND_UNIQUE_AUTHORS,
    GET_IN_NETWORK_POSTS_MAX_RESULTS, IN_FLIGHT_REQUESTS, REJECTED_REQUESTS, Timer,
};
use crate::posts::post_store::PostStore;
use crate::strato_client::StratoClient;

pub struct ThunderServiceImpl {
    /// 用于按用户ID检索帖子的 PostStore
    post_store: Arc<PostStore>,
    /// 用于在未提供时获取关注列表的 StratoClient
    strato_client: Arc<StratoClient>,
    /// 用于限制并发请求并防止过载的信号量
    request_semaphore: Arc<Semaphore>,
}

impl ThunderServiceImpl {
    pub fn new(
        post_store: Arc<PostStore>,
        strato_client: Arc<StratoClient>,
        max_concurrent_requests: usize,
    ) -> Self {
        info!(
            "Initializing ThunderService with max_concurrent_requests={}",
            max_concurrent_requests
        );
        Self {
            post_store,
            strato_client,
            request_semaphore: Arc::new(Semaphore::new(max_concurrent_requests)),
        }
    }

    /// 为此服务创建 gRPC 服务器
    pub fn server(self) -> InNetworkPostsServiceServer<Self> {
        InNetworkPostsServiceServer::new(self)
            .accept_compressed(tonic::codec::CompressionEncoding::Zstd)
            .send_compressed(tonic::codec::CompressionEncoding::Zstd)
    }

    /// 分析找到的帖子，计算统计信息并报告指标
    /// `stage` 参数用作标签以区分不同阶段（例如，"post_store"、"scored"）
    fn analyze_and_report_post_statistics(posts: &[LightPost], stage: &str) {
        if posts.is_empty() {
            debug!("[{}] No posts found for analysis", stage);
            return;
        }

        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;

        // 距离最新帖子的时间
        let time_since_most_recent = posts
            .iter()
            .map(|post| post.created_at)
            .max()
            .map(|most_recent| now - most_recent);

        // 距离最旧帖子的时间
        let time_since_oldest = posts
            .iter()
            .map(|post| post.created_at)
            .min()
            .map(|oldest| now - oldest);

        // 统计回复与原始帖子
        let reply_count = posts.iter().filter(|post| post.is_reply).count();
        let original_count = posts.len() - reply_count;

        // 唯一作者
        let unique_authors: HashSet<_> = posts.iter().map(|post| post.author_id).collect();
        let unique_author_count = unique_authors.len();

        // 使用阶段标签报告指标
        if let Some(freshness) = time_since_most_recent {
            GET_IN_NETWORK_POSTS_FOUND_FRESHNESS_SECONDS
                .with_label_values(&[stage])
                .observe(freshness as f64);
        }

        if let (Some(oldest), Some(newest)) = (time_since_oldest, time_since_most_recent) {
            let time_range = oldest - newest;
            GET_IN_NETWORK_POSTS_FOUND_TIME_RANGE_SECONDS
                .with_label_values(&[stage])
                .observe(time_range as f64);
        }

        let reply_ratio = reply_count as f64 / posts.len() as f64;
        GET_IN_NETWORK_POSTS_FOUND_REPLY_RATIO
            .with_label_values(&[stage])
            .observe(reply_ratio);

        GET_IN_NETWORK_POSTS_FOUND_UNIQUE_AUTHORS
            .with_label_values(&[stage])
            .observe(unique_author_count as f64);

        if unique_author_count > 0 {
            let posts_per_author = posts.len() as f64 / unique_author_count as f64;
            GET_IN_NETWORK_POSTS_FOUND_POSTS_PER_AUTHOR
                .with_label_values(&[stage])
                .observe(posts_per_author);
        }

        // 使用阶段标签记录统计信息
        debug!(
            "[{}] Post statistics: total={}, original={}, replies={}, unique_authors={}, posts_per_author={:.2}, reply_ratio={:.2}, time_since_most_recent={:?}s, time_range={:?}s",
            stage,
            posts.len(),
            original_count,
            reply_count,
            unique_author_count,
            if unique_author_count > 0 {
                posts.len() as f64 / unique_author_count as f64
            } else {
                0.0
            },
            reply_ratio,
            time_since_most_recent,
            if let (Some(o), Some(n)) = (time_since_oldest, time_since_most_recent) {
                Some(o - n)
            } else {
                None
            }
        );
    }
}

#[tonic::async_trait]
impl InNetworkPostsService for ThunderServiceImpl {
    /// 从网络中的用户获取帖子
    async fn get_in_network_posts(
        &self,
        request: Request<GetInNetworkPostsRequest>,
    ) -> Result<Response<GetInNetworkPostsResponse>, Status> {
        // 尝试获取信号量许可而不阻塞
        // 如果已达到容量，立即使用 RESOURCE_EXHAUSTED 拒绝
        let _permit = match self.request_semaphore.try_acquire() {
            Ok(permit) => {
                IN_FLIGHT_REQUESTS.inc();
                permit
            }
            Err(_) => {
                REJECTED_REQUESTS.inc();
                return Err(Status::resource_exhausted(
                    "Server at capacity, please retry",
                ));
            }
        };

        // 使用守卫在请求完成时递减 in_flight_requests
        struct InFlightGuard;
        impl Drop for InFlightGuard {
            fn drop(&mut self) {
                IN_FLIGHT_REQUESTS.dec();
            }
        }
        let _in_flight_guard = InFlightGuard;

        // 启动总延迟计时器
        let _total_timer = Timer::new(GET_IN_NETWORK_POSTS_DURATION.clone());

        let req = request.into_inner();

        if req.debug {
            info!(
                "Received GetInNetworkPosts request: user_id={}, following_count={}, exclude_tweet_ids={}",
                req.user_id,
                req.following_user_ids.len(),
                req.exclude_tweet_ids.len(),
            );
        }

        // 如果 following_user_id 列表为空，从 Strato 获取
        let following_user_ids = if req.following_user_ids.is_empty() && req.debug {
            info!(
                "Following list is empty, fetching from Strato for user {}",
                req.user_id
            );

            match self
                .strato_client
                .fetch_following_list(req.user_id as i64, MAX_INPUT_LIST_SIZE as i32)
                .await
            {
                Ok(following_list) => {
                    info!(
                        "Fetched {} following users from Strato for user {}",
                        following_list.len(),
                        req.user_id
                    );
                    following_list.into_iter().map(|id| id as u64).collect()
                }
                Err(e) => {
                    warn!(
                        "Failed to fetch following list from Strato for user {}: {}",
                        req.user_id, e
                    );
                    return Err(Status::internal(format!(
                        "Failed to fetch following list: {}",
                        e
                    )));
                }
            }
        } else {
            req.following_user_ids
        };

        // 记录请求参数的指标
        GET_IN_NETWORK_POSTS_FOLLOWING_SIZE.observe(following_user_ids.len() as f64);
        GET_IN_NETWORK_POSTS_EXCLUDED_SIZE.observe(req.exclude_tweet_ids.len() as f64);

        // 启动不包含 strato 调用的延迟计时器
        let _processing_timer = Timer::new(GET_IN_NETWORK_POSTS_DURATION_WITHOUT_STRATO.clone());

        // 如果未指定，则使用默认 max_results
        let max_results = if req.max_results > 0 {
            req.max_results as usize
        } else if req.is_video_request {
            MAX_VIDEOS_TO_RETURN
        } else {
            MAX_POSTS_TO_RETURN
        };
        GET_IN_NETWORK_POSTS_MAX_RESULTS.observe(max_results as f64);

        // 将 following_user_ids 和 exclude_tweet_ids 限制为前 K 个条目
        let following_count = following_user_ids.len();
        if following_count > MAX_INPUT_LIST_SIZE {
            warn!(
                "Limiting following_user_ids from {} to {} entries for user {}",
                following_count, MAX_INPUT_LIST_SIZE, req.user_id
            );
        }
        let following_user_ids: Vec<u64> = following_user_ids
            .into_iter()
            .take(MAX_INPUT_LIST_SIZE)
            .collect();

        let exclude_count = req.exclude_tweet_ids.len();
        if exclude_count > MAX_INPUT_LIST_SIZE {
            warn!(
                "Limiting exclude_tweet_ids from {} to {} entries for user {}",
                exclude_count, MAX_INPUT_LIST_SIZE, req.user_id
            );
        }
        let exclude_tweet_ids: Vec<u64> = req
            .exclude_tweet_ids
            .into_iter()
            .take(MAX_INPUT_LIST_SIZE)
            .collect();

        // 克隆在 spawn_blocking 内部需要的 Arc 引用
        let post_store = Arc::clone(&self.post_store);
        let request_user_id = req.user_id as i64;

        // 使用 spawn_blocking 避免阻塞 tokio 的异步运行时
        let proto_posts = tokio::task::spawn_blocking(move || {
            // 创建排除推文ID集合以高效过滤之前见过的帖子
            let exclude_tweet_ids: HashSet<i64> =
                exclude_tweet_ids.iter().map(|&id| id as i64).collect();

            let start_time = Instant::now();

            // 获取已关注用户的所有帖子（原始 + 次要）
            let all_posts: Vec<LightPost> = if req.is_video_request {
                post_store.get_videos_by_users(
                    &following_user_ids,
                    &exclude_tweet_ids,
                    start_time,
                    request_user_id,
                )
            } else {
                post_store.get_all_posts_by_users(
                    &following_user_ids,
                    &exclude_tweet_ids,
                    start_time,
                    request_user_id,
                )
            };

            // 在查询 post_store 后分析帖子并报告统计信息
            ThunderServiceImpl::analyze_and_report_post_statistics(&all_posts, "retrieved");

            let scored_posts = score_recent(all_posts, max_results);

            // 在打分后分析帖子并报告统计信息
            ThunderServiceImpl::analyze_and_report_post_statistics(&scored_posts, "scored");

            scored_posts
        })
        .await
        .map_err(|e| Status::internal(format!("Failed to process posts: {}", e)))?;

        if req.debug {
            info!(
                "Returning {} posts for user {}",
                proto_posts.len(),
                req.user_id
            );
        }

        // 记录返回的帖子数量
        GET_IN_NETWORK_POSTS_COUNT.observe(proto_posts.len() as f64);

        let response = GetInNetworkPostsResponse { posts: proto_posts };

        Ok(Response::new(response))
    }
}

/// 按新鲜度对帖子打分（created_at 时间戳，较新的帖子在前）
fn score_recent(mut light_posts: Vec<LightPost>, max_results: usize) -> Vec<LightPost> {
    light_posts.sort_unstable_by_key(|post| Reverse(post.created_at));

    // 限制为最大结果数
    light_posts.into_iter().take(max_results).collect()
}
