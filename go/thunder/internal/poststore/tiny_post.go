package poststore

// TinyPost represents a minimal post reference stored in user timelines
// (only ID and timestamp)
type TinyPost struct {
	PostID    int64
	CreatedAt int64
}

// NewTinyPost creates a new TinyPost from a post ID and creation timestamp
func NewTinyPost(postID, createdAt int64) *TinyPost {
	return &TinyPost{
		PostID:    postID,
		CreatedAt: createdAt,
	}
}
