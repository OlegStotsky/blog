package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/OlegStotsky/goflatdb"
	"go.uber.org/zap"

	"blog"
)

func main() {
	dbDirF := flag.String("db-dir", "", "path to db")

	flag.Parse()

	if *dbDirF == "" {
		fmt.Println("comment-dir flag can't be empty")

		os.Exit(1)
	}

	logger, _ := zap.NewProduction()

	db, err := goflatdb.NewFlatDB(*dbDirF, logger)
	if err != nil {
		fmt.Println("error creating db", err)

		os.Exit(1)
	}

	commentsCollection, err := goflatdb.NewFlatDBCollection[blog.Comment](db, blog.CommentsCollectionName, logger)
	if err != nil {
		fmt.Println("error creating comments collection", err)

		os.Exit(1)
	}

	if err := commentsCollection.Init(); err != nil {
		fmt.Println("error initing comments collection", err)

		os.Exit(1)
	}

	commentService, err := blog.NewCommentService(commentsCollection)
	if err != nil {
		fmt.Println("error creating comment service", err)

		os.Exit(1)
	}

	postsCollection, err := goflatdb.NewFlatDBCollection[blog.Post](db, blog.PostsCollectionName, logger)
	if err != nil {
		fmt.Println("error creating posts collection", err)

		os.Exit(1)
	}

	if err := postsCollection.Init(); err != nil {
		fmt.Println("error initing comments collection", err)

		os.Exit(1)
	}

	postsService := blog.NewPostService(postsCollection)

	server, err := blog.NewServer("localhost:3000", commentService, postsService)
	if err != nil {
		panic(err)
	}

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
