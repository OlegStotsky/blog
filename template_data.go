package blog

type PostTemplateData struct {
	Title    string
	Content  string
	Comments []CommentTemplateData
}

type CommentTemplateData struct {
	Author  string
	Date    string
	Comment string
}

type PostsTemplateData struct {
	Name string
	ID   uint64
	Date string
}
