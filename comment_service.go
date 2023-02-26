package blog

import (
	"fmt"
	"time"

	"github.com/OlegStotsky/goflatdb"
)

const CommentsCollectionName = "comments"

type CommentService struct {
	commentsCollection *goflatdb.FlatDBCollection[Comment]
}

type Comment struct {
	PostID  uint64    `json:"post_id,omitempty"`
	Author  string    `json:"author,omitempty"`
	Date    time.Time `json:"date"`
	Comment string    `json:"comment,omitempty"`
}

func NewCommentService(commentsCollection *goflatdb.FlatDBCollection[Comment]) (*CommentService, error) {
	return &CommentService{commentsCollection: commentsCollection}, nil
}

func (c *CommentService) SaveComment(comment *Comment) error {
	fmt.Printf("saving comment %+v\n", comment)

	res, err := c.commentsCollection.Insert(comment)
	if err != nil {
		return fmt.Errorf("error saving comment: %w", err)
	}

	fmt.Println("successfully created saved comment", res.ID)

	return nil
}

func (c *CommentService) GetComments(postID uint64) ([]goflatdb.FlatDBModel[Comment], error) {
	comments, err := c.commentsCollection.QueryBuilder().Where("PostID", "=", postID).Execute()
	if err != nil {
		return nil, fmt.Errorf("error getting comments: %w", err)
	}

	return comments, nil
}
