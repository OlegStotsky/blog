package blog

type PostTemplateData struct {
	Title   string
	Content string
}

type PostsTemplateData struct {
	Name string
	ID   uint64
	Date string
}
