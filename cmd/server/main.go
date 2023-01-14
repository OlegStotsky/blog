package main

import "blog"

func main() {
	server, err := blog.NewServer("localhost:3000")
	if err != nil {
		panic(err)
	}

	err = server.RenderMarkdownToHTML()
	if err != nil {
		panic(err)
	}

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
