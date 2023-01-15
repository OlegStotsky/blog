package blog

import (
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
}

func NewServer(addr string) (*Server, error) {
	server := &Server{addr: addr}

	r := mux.NewRouter()
	r.HandleFunc("/", server.Posts).Methods(http.MethodGet)
	r.HandleFunc("/posts/{postName}", server.Post).Methods(http.MethodGet)
	r.HandleFunc("/static/{fileName}", server.StaticHandler).Methods(http.MethodGet)

	srv := http.Server{Addr: addr}

	srv.Handler = r

	server.server = &srv

	postsTemplate, err := template.ParseFiles("./templates/posts.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing posts template: %w", err)
	}
	server.postsTemplate = postsTemplate

	postTemplate, err := template.ParseFiles("./templates/post.html")
	if err != nil {
		return nil, fmt.Errorf("error parsing post template: %w", err)
	}
	server.postTemplate = postTemplate

	return server, nil
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
