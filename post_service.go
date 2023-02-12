package blog

import (
	"fmt"
	"time"

	"github.com/OlegStotsky/goflatdb"
)

const PostsCollectionName = "posts"

type Post struct {
	Name    string    `json:"name"`
	Date    time.Time `json:"date"`
	Content string    `json:"content"`
}

type PostService struct {
	postCollection *goflatdb.FlatDBCollection[Post]
}

func NewPostService(col *goflatdb.FlatDBCollection[Post]) *PostService {
	return &PostService{postCollection: col}
}

func (c *PostService) AddPost(post *Post) error {
	res, err := c.postCollection.Insert(post)
	if err != nil {
		return fmt.Errorf("error saving post: %w", err)
	}

	fmt.Println("successfully saved post", res.ID)

	return nil
}

func (c *PostService) GetPosts() ([]goflatdb.FlatDBModel[Post], error) {
	posts, err := c.postCollection.QueryBuilder().Select().Execute()
	if err != nil {
		return nil, fmt.Errorf("error getting posts: %w", err)
	}

	return posts, nil
}

func (c *PostService) GetPost(id uint64) (goflatdb.FlatDBModel[Post], error) {
	post, err := c.postCollection.GetByID(id)
	if err != nil {
		return goflatdb.FlatDBModel[Post]{}, fmt.Errorf("error getting posts: %w", err)
	}

	return post, nil
}
