package config

// Constants for Thunder service configuration
const (
	// Maximum input list size
	MAX_INPUT_LIST_SIZE = 10000

	// Maximum posts to return
	MAX_POSTS_TO_RETURN = 1000

	// Maximum videos to return
	MAX_VIDEO_POSTS_TO_RETURN = 500

	// Maximum original posts per author
	MAX_ORIGINAL_POSTS_PER_AUTHOR = 10

	// Maximum reply posts per author
	MAX_REPLY_POSTS_PER_AUTHOR = 5

	// Maximum video posts per author
	MAX_VIDEO_POSTS_PER_AUTHOR = 5

	// Maximum tiny posts per user scan
	MAX_TINY_POSTS_PER_USER_SCAN = 1000

	// Delete event key (special user ID for tracking deletions)
	DELETE_EVENT_KEY = -1
)
