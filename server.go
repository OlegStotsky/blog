package blog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/OlegStotsky/goflatdb"
	"github.com/gomarkdown/markdown"
	"github.com/gorilla/mux"
)

const PostDateLayout = "2006-01-02-15:04"

type Server struct {
	addr   string
	server *http.Server

	debugMode bool

	postsTemplate *template.Template
	postTemplate  *template.Template
	aboutTemplate *template.Template

	commentService *CommentService
	postService    *PostService
}

func NewServer(addr string, commentService *CommentService, postService *PostService) (*Server, error) {
	server := &Server{
		addr:           addr,
		debugMode:      true,
		commentService: commentService,
		postService:    postService,
	}

	fs := http.FileServer(http.Dir("./static"))

	r := mux.NewRouter()
	r.HandleFunc("/", server.PostsHandler).Methods(http.MethodGet)
	r.HandleFunc("/posts/{postID}", server.PostHandler).Methods(http.MethodGet)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	r.HandleFunc("/about", server.AboutHandler).Methods(http.MethodGet)
	r.HandleFunc("/posts/{postID}/comments", server.CommentHandler).Methods(http.MethodPost)

	srv := http.Server{Addr: addr}

	srv.Handler = r

	server.server = &srv

	postsTemplate, err := template.ParseFiles("./templates/posts.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing posts template: %w", err)
	}
	if err := addAllPageTemplates(postsTemplate); err != nil {
		return nil, err
	}

	server.postsTemplate = postsTemplate

	postTemplate, err := template.ParseFiles("./templates/post.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing post template: %w", err)
	}
	if err := addAllPageTemplates(postTemplate); err != nil {
		return nil, err
	}

	server.postTemplate = postTemplate

	aboutTemplate, err := template.ParseFiles("./templates/about.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing about template: %w", err)
	}
	if err := addAllPageTemplates(aboutTemplate); err != nil {
		return nil, err
	}

	server.aboutTemplate = aboutTemplate

	return server, nil
}

func addAllPageTemplates(t *template.Template) error {
	if _, err := t.New("navbar").ParseFiles("./templates/navbar.html"); err != nil {
		return fmt.Errorf("error parsing navbar template: %w", err)
	}
	if _, err := t.New("google-analytics").ParseFiles("./templates/google-analytics.html"); err != nil {
		return fmt.Errorf("error parsing google analytics template: %w", err)
	}
	if _, err := t.New("footer").ParseFiles("./templates/footer.html"); err != nil {
		return fmt.Errorf("error parsing footer template: %w", err)
	}

	return nil
}

func (c *Server) ListenAndServe() error {
	return c.server.ListenAndServe()
}

func (c *Server) StaticHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	fileName := vars["fileName"]

	fmt.Println("static handler for file", fileName)

	f, err := os.Open(filepath.Join("./static", fileName))
	if err != nil {
		fmt.Println("error serving static file", fileName, err)

		rw.WriteHeader(500)

		return
	}

	if _, err := io.Copy(rw, f); err != nil {
		fmt.Println("error serving static file", fileName, err)

		return
	}
}

func (c *Server) AboutHandler(rw http.ResponseWriter, r *http.Request) {
	fmt.Println("got new about request")

	if err := c.aboutTemplate.Execute(rw, nil); err != nil {
		fmt.Println("error serving about", err)

		return
	}
}

func (c *Server) PostsHandler(rw http.ResponseWriter, r *http.Request) {
	fmt.Println("got new posts request")

	posts, err := c.postService.GetPosts()
	if err != nil {
		fmt.Println("error listing posts", err)
		rw.WriteHeader(500)
		return
	}

	postsViewData := []PostsTemplateData{}
	for _, post := range posts {
		postsViewData = append(postsViewData, PostToPostsViewData(&post))
	}

	if err := c.postsTemplate.Execute(rw, postsViewData); err != nil {
		fmt.Println("error executing posts template", err)
		rw.WriteHeader(500)
		return
	}

	fmt.Println("servers posts successfully")
}

func PostToPostsViewData(post *goflatdb.FlatDBModel[Post]) PostsTemplateData {
	return PostsTemplateData{
		Name: post.Data.Name,
		ID:   post.ID,
		Date: post.Data.Date.Format(PostDateLayout),
	}
}

func CommentToCommentViewData(comment *goflatdb.FlatDBModel[Comment]) CommentTemplateData {
	return CommentTemplateData{
		Author:  comment.Data.Author,
		Date:    comment.Data.Date.Format(PostDateLayout),
		Comment: comment.Data.Comment,
	}
}

func PostToPostViewData(post *goflatdb.FlatDBModel[Post], comments []CommentTemplateData) PostTemplateData {
	return PostTemplateData{
		Title:    post.Data.Name,
		Content:  post.Data.Content,
		Comments: comments,
	}
}

func (c *Server) PostHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	postID := vars["postID"]

	fmt.Println("got new post request", postID)

	postIDInt, err := strconv.ParseUint(postID, 10, 64)
	if err != nil {
		writeErrorResponse(400, ErrorBadPostID, "invalid post id", rw)

		return
	}

	post, err := c.postService.GetPost(postIDInt)
	if err != nil {
		fmt.Println("error getting post", err)

		rw.WriteHeader(500)

		return
	}

	comments, err := c.commentService.GetComments(postIDInt)
	if err != nil {
		fmt.Println("error getting comments for post", postIDInt, err)
	}
	commentsViewData := []CommentTemplateData{}
	for _, comment := range comments {
		commentsViewData = append(commentsViewData, CommentToCommentViewData(&comment))
	}
	fmt.Println(comments)

	if err := c.postTemplate.Execute(rw, PostToPostViewData(&post, commentsViewData)); err != nil {
		fmt.Println("error executing post template", err)

		return
	}
}

type CommentRequest struct {
	Author  string `json:"author"`
	Comment string `json:"comment"`
}

type CommentResponse struct {
	Author  string `json:"author"`
	Comment string `json:"comment"`
}

type ErrorResponse struct {
	ErrorCode    int    `json:"code"`
	ErrorMessage string `json:"message"`
}

func (c *Server) CommentHandler(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(400)
	return

	vars := mux.Vars(r)

	postID := vars["postID"]

	fmt.Println("handling comment request for post", postID)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("error handling comment request", err)

		rw.WriteHeader(500)

		return
	}

	commentRequest := CommentRequest{}
	if err := json.Unmarshal(body, &commentRequest); err != nil {
		fmt.Println("error handling comment")

		rw.WriteHeader(500)

		return
	}

	if len(commentRequest.Author) < 1 {
		writeErrorResponse(http.StatusBadRequest, ErrorBadCommentAuthorLen, "author should be at least 1 character long", rw)

		return

	}

	if len(commentRequest.Author) > 100 {
		writeErrorResponse(http.StatusBadRequest, ErrorBadCommentAuthorLen, "author should be under 100 characters", rw)

		return
	}

	if len(commentRequest.Comment) > 2000 {
		writeErrorResponse(http.StatusBadRequest, ErrorBadCommentLen, "comment length should be under 2000 characters", rw)

		return
	}

	if len(commentRequest.Comment) < 1 {
		writeErrorResponse(http.StatusBadRequest, ErrorBadCommentLen, "comment length should be > 1", rw)

		return
	}

	if len(commentRequest.Comment) < 1 {
		writeErrorResponse(http.StatusBadRequest, ErrorBadCommentLen, "comment length should be more than 1 character", rw)

		return
	}

	postIDInt, err := strconv.ParseUint(postID, 10, 64)
	if err != nil {
		writeErrorResponse(400, ErrorBadPostID, "invalid post id", rw)

		return
	}

	err = c.commentService.SaveComment(&Comment{
		PostID:  postIDInt,
		Author:  commentRequest.Author,
		Date:    time.Now(),
		Comment: commentRequest.Comment,
	})
	if err != nil {
		fmt.Println("error saving comment", err)

		rw.WriteHeader(500)

		return
	}

	resp := CommentResponse{
		Author:  commentRequest.Author,
		Comment: commentRequest.Comment,
	}
	respBytes, err := json.Marshal(&resp)
	if err != nil {
		fmt.Println("error marshalling comment response", err)

		rw.WriteHeader(500)

		return
	}

	rw.Write(respBytes)
}

func writeErrorResponse(statusCode int, errorCode int, errorMessage string, rw http.ResponseWriter) error {
	rw.WriteHeader(statusCode)

	e := ErrorResponse{
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}

	bytes, err := json.Marshal(&e)
	if err != nil {
		return fmt.Errorf("error marshalling error resp: %w", err)
	}

	rw.Write(bytes)

	return nil
}

func RenderMarkdownToHTML(markdownBytes []byte) []byte {
	return markdown.ToHTML(markdownBytes, nil, nil)
}
