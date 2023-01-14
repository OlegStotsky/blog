package main

import (
	"fmt"
	"os"
	"time"

	"blog"
)

func main() {
	fmt.Println("enter post name")

	var postName string
	_, err := fmt.Scanln(&postName)
	if err != nil {
		panic(err)
	}

	postInfo := blog.PostInfo{
		Date: time.Now(),
		Name: postName,
		Dir:  "./posts",
	}

	path := postInfo.ToFilePath("md")

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	fmt.Println("created post file", f.Name())
}
