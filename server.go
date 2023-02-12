package blog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

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
}

func NewServer(addr string, commentService *CommentService) (*Server, error) {
	server := &Server{
		addr:           addr,
		debugMode:      true,
		commentService: commentService,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", server.Posts).Methods(http.MethodGet)
	r.HandleFunc("/posts/{postName}", server.Post).Methods(http.MethodGet)
	r.HandleFunc("/static/{fileName}", server.StaticHandler).Methods(http.MethodGet)
	r.HandleFunc("/about", server.AboutHandler).Methods(http.MethodGet)
	r.HandleFunc("/posts/{postName}/comments", server.CommentHandler).Methods(http.MethodPost)

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

func (c *Server) Posts(rw http.ResponseWriter, r *http.Request) {
	fmt.Println("got new posts request")

	posts, err := filepath.Glob("./posts/*.md")
	if err != nil {
		fmt.Println("error listing posts", err)
		rw.WriteHeader(500)
		return
	}

	postsTemplateDatas := []PostsTemplateData{}
	for _, post := range posts {
		file, err := os.Open(post)
		if err != nil {
			fmt.Println("error reading post", err)
			rw.WriteHeader(500)
			return
		}

		postInfo, err := ParsePostInfo(file.Name())
		if err != nil {
			fmt.Println("error parsing post", err)
			rw.WriteHeader(500)
			return
		}

		postsTemplateData := PostsTemplateData{
			FileName: postInfo.ToFileName("html"),
			PostName: postInfo.Name,
			PostDate: postInfo.Date.Format("Jan 02 15:04 2006"),
		}

		postsTemplateDatas = append(postsTemplateDatas, postsTemplateData)
	}

	if err := c.postsTemplate.Execute(rw, postsTemplateDatas); err != nil {
		fmt.Println("error executing posts template", err)
		rw.WriteHeader(500)
		return
	}

	fmt.Println("servers posts successfully")
}

func (c *Server) Post(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	postName := vars["postName"]

	fmt.Println("got new post request", postName)

	postFilePath := fmt.Sprintf("./posts/%s", postName)
	postInfo, err := ParsePostInfo(postFilePath)
	if err != nil {
		fmt.Println("error parsing post file info", err)

		rw.WriteHeader(500)

		return
	}

	_, err = RenderMarkdownPostToHTML(postInfo.ToFilePath("md"))
	if err != nil {
		fmt.Println("could render missing html file", postFilePath, err)

		rw.WriteHeader(500)
	}

	f, err := os.Open(postFilePath)
	if err != nil {
		fmt.Println("error opening post file", err, postFilePath)
		fmt.Println("will try to render markdown file with the same name", postFilePath)

		_, err := RenderMarkdownPostToHTML(postInfo.ToFilePath("md"))
		if err != nil {
			fmt.Println("could render missing html file", postFilePath, err)

			rw.WriteHeader(500)

			return
		} else {
			fmt.Println("successfully rendered missing html file", postFilePath)

			f, err = os.Open(postFilePath)
			if err != nil {
				fmt.Println("couldn't reopen missing html file after rendering markdown", postFilePath)

				rw.WriteHeader(500)

				return
			}
		}
	}

	postContent, err := io.ReadAll(f)
	if err != nil {
		fmt.Println("error reading post file", err)
		rw.WriteHeader(500)
		return
	}

	postTemplateData := &PostTemplateData{
		Title:   postInfo.Name,
		Content: string(postContent),
	}

	if err := c.postTemplate.Execute(rw, postTemplateData); err != nil {
		fmt.Println("error executing post template", err)

		return
	}
}

func (c *Server) RenderMarkdownToHTML() error {
	posts, err := filepath.Glob("./posts/*.md")
	if err != nil {
		return fmt.Errorf("glob error: %w", err)
	}

	fmt.Println("got posts", posts)

	for _, post := range posts {
		postInfo, err := RenderMarkdownPostToHTML(post)
		if err != nil {
			return err
		}

		fmt.Println("created rendered html for post", postInfo.Name)
	}

	return nil
}

type CommentRequest struct {
	Author  string `json:"author"`
	Comment string `json:"comment"`
}

type ErrorResponse struct {
	ErrorCode    int    `json:"code"`
	ErrorMessage string `json:"message"`
}

func (c *Server) CommentHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	postName := vars["postName"]

	fmt.Println("handling comment request for post", postName)

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

	if len(commentRequest.Author) > 100 {
		writeErrorResponse(http.StatusBadRequest, ErrorBadCommentAuthorLen, "author should be under 100 characters", rw)

		return
	}

	if len(commentRequest.Author) > 2000 {
		writeErrorResponse(http.StatusBadRequest, ErrorBadCommentLen, "comment length should be under 2000 characters", rw)

		return
	}

	err = c.commentService.SaveComment(&Comment{
		Post:    postName,
		Author:  commentRequest.Author,
		Date:    time.Now(),
		Comment: commentRequest.Comment,
	})
	if err != nil {
		fmt.Println("error saving comment", err)

		rw.WriteHeader(500)

		return
	}

	rw.WriteHeader(200)
	fmt.Println("successfully saved comment")
}

func writeErrorResponse(statusCode int, errorCode int, errorMessage string, rw http.ResponseWriter) error {
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

func RenderMarkdownPostToHTML(postFilePath string) (PostInfo, error) {
	file, err := os.Open(postFilePath)
	if err != nil {
		return PostInfo{}, fmt.Errorf("error reading post: %w", err)
	}

	postInfo, err := ParsePostInfo(file.Name())
	if err != nil {
		return PostInfo{}, err
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return PostInfo{}, fmt.Errorf("error reading file %s: %w", file.Name(), err)
	}

	markdownContent := RenderMarkdownToHTML(content)

	outFilePath := postInfo.ToFilePath("html")
	res, err := os.Create(outFilePath)
	if err != nil {
		return PostInfo{}, fmt.Errorf("error creating rendered markdown file %s: %w", outFilePath, err)
	}

	_, err = res.Write(markdownContent)
	if err != nil {
		return PostInfo{}, fmt.Errorf("error writing html content to %s: %w", outFilePath, err)
	}

	return postInfo, nil
}

func RenderMarkdownToHTML(markdownBytes []byte) []byte {
	return markdown.ToHTML(markdownBytes, nil, nil)
}

type PostInfo struct {
	Date time.Time
	Name string
	Dir  string
}

func (c *PostInfo) ToFilePath(extension string) string {
	date := c.Date.Format(PostDateLayout)

	fileName := fmt.Sprintf("%s_%s.%s", date, c.Name, extension)

	return filepath.Join(c.Dir, fileName)
}

func (c *PostInfo) ToFileName(extension string) string {
	date := c.Date.Format(PostDateLayout)

	return fmt.Sprintf("%s_%s.%s", date, c.Name, extension)
}

// parses post info of type yyyyy-mm-dd-hh:mm_my-post-name.ext
func ParsePostInfo(filePath string) (PostInfo, error) {
	dir := filepath.Dir(filePath)
	name := filepath.Base(filePath)

	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return PostInfo{}, fmt.Errorf("error parsing post filename %s: got more than 2 parts", filePath)
	}

	file := parts[0]

	parts2 := strings.Split(file, "_")
	if len(parts) != 2 {
		return PostInfo{}, fmt.Errorf("error parsing post filename %s: got more than 2 parts", filePath)
	}

	date, err := time.Parse(PostDateLayout, parts2[0])
	if err != nil {
		return PostInfo{}, fmt.Errorf("error parsing date from file %s: %w", filePath, err)
	}

	postName := parts2[1]

	return PostInfo{
		Date: date,
		Name: postName,
		Dir:  dir,
	}, nil
}
