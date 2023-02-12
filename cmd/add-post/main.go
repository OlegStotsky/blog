package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/OlegStotsky/goflatdb"
	"go.uber.org/zap"

	"blog"
)

func main() {
	logger, _ := zap.NewProduction()

	db, err := goflatdb.NewFlatDB("/var/lib/blog", logger)
	if err != nil {
		fmt.Println("error opening db", err)

		return
	}

	col, err := goflatdb.NewFlatDBCollection[blog.Post](db, "posts", logger)
	if err != nil {
		fmt.Println("error opening posts collection", err)

		return
	}

	fmt.Println("enter post name")

	reader := bufio.NewScanner(os.Stdin)
	ok := reader.Scan()
	if !ok {
		fmt.Println("no input")
		return
	}
	postName := reader.Text()

	fmt.Println("enter post filepath")

	ok = reader.Scan()
	if !ok {
		fmt.Println("no input")
		return
	}
	postPath := reader.Text()

	f, err := os.Open(postPath)
	if err != nil {
		fmt.Println("error opening post file", err)

		return
	}

	content, err := io.ReadAll(f)
	if err != nil {
		fmt.Println("error reading post file", err)

		return
	}

	markdown := blog.RenderMarkdownToHTML(content)

	res, err := col.Insert(&blog.Post{
		Name:    postName,
		Date:    time.Now(),
		Content: string(markdown),
	})
	if err != nil {
		fmt.Println("error inserting post", err)

		return
	}

	fmt.Println("successfully inserted post", res.ID)
}
