package blog

import (
	"fmt"
	"time"

	"github.com/OlegStotsky/goflatdb"
)

type CommentService struct {
	commentsCollection *goflatdb.FlatDBCollection[Comment]
}

type Comment struct {
	Post    string    `json:"post,omitempty"`
	Author  string    `json:"author,omitempty"`
	Date    time.Time `json:"date"`
	Comment string    `json:"comment,omitempty"`
}

func NewCommentService(commentsCollection *goflatdb.FlatDBCollection[Comment]) (*CommentService, error) {
	return &CommentService{commentsCollection: commentsCollection}, nil
}

func (c *CommentService) SaveComment(comment *Comment) error {
	res, err := c.commentsCollection.Insert(comment)
	if err != nil {
		return fmt.Errorf("error saving comment: %w", err)
	}

	fmt.Println("successfully created saved comment", res.ID)

	return nil
}
