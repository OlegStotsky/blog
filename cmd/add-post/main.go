package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"blog"
)

func main() {
	fmt.Println("enter post name")

	reader := bufio.NewScanner(os.Stdin)
	ok := reader.Scan()
	if !ok {
		fmt.Println("no input")
		return
	}

	postName := reader.Text()

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
