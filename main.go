// Copyright 2012 Arne Roomann-Kurrik
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/knieriem/markdown"
	"github.com/kurrik/fauxfile"
	"io"
	"launchpad.net/goyaml"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Arguments, passed to the main executable.
type Args struct {
	src       string
	dst       string
	posts     string
	templates string
	static    string
	config    string
}

func DefaultArgs() *Args {
	return &Args{
		src:       "src",
		dst:       "dst",
		posts:     "posts",
		templates: "templates",
		static:    "static",
		config:    "config.yaml",
	}
}

// Master Control Program.
type GhostWriter struct {
	args      *Args
	fs        fauxfile.Filesystem
	log       *log.Logger
	site      *Site
	templates *template.Template
}

// Creates a new GhostWriter.
func NewGhostWriter(fs fauxfile.Filesystem, args *Args) *GhostWriter {
	gw := &GhostWriter{
		args: args,
		fs:   fs,
		log:  log.New(os.Stderr, "", log.LstdFlags),
		site: &Site{
			Posts: make(map[string]*Post),
		},
	}
	return gw
}

// Parses the src directory, rendering into dst as needed.
func (gw *GhostWriter) Process() (err error) {
	if err = gw.fs.MkdirAll(gw.args.dst, 0755); err != nil {
		return
	}
	if err = gw.parseSiteMeta(); err != nil {
		return
	}
	if err = gw.parseTemplates(); err != nil {
		return
	}
	if err = gw.parsePosts(); err != nil {
		return
	}
	if err = gw.renderMisc(); err != nil {
		return
	}
	return
}

// Copies the file at path src to path dst.
// Returns the number of bytes written or an error if it occurred.
func (gw *GhostWriter) copyFile(src string, dst string) (n int64, err error) {
	var (
		fdst fauxfile.File
		fsrc fauxfile.File
		fi   os.FileInfo
	)
	if fsrc, err = gw.fs.Open(src); err != nil {
		return
	}
	defer fsrc.Close()
	if fdst, err = gw.fs.Create(dst); err != nil {
		return
	}
	defer fdst.Close()
	if fi, err = fsrc.Stat(); err != nil {
		return
	}
	fdst.Chmod(fi.Mode())
	_, err = io.Copy(fdst, fsrc)
	return
}

// Returns a copy of the global template with the supplied template merged in.
func (gw *GhostWriter) mergeTemplate(t *template.Template) (out *template.Template, err error) {
	if out, err = gw.templates.Clone(); err != nil {
		return
	}
	for _, tmpl := range t.Templates() {
		ptr := out.Lookup(tmpl.Name())
		if ptr == nil {
			ptr = out.New(tmpl.Name())
		}
		(*ptr) = *tmpl
	}
	return
}

// Parses a post meta file at the given path.
// Returns a pointer to a populated PostMeta object or an error if it failed.
func (gw *GhostWriter) parsePostMeta(path string) (meta *PostMeta, err error) {
	src := filepath.Join(gw.args.src, path)
	gw.log.Printf("Parsing site meta %v\n", src)
	meta = &PostMeta{}
	err = gw.unyaml(src, meta)
	return
}

// Parses posts under the supplied path and populates gw.site.Posts.
func (gw *GhostWriter) parsePosts() (err error) {
	var (
		name   = gw.args.posts
		src    = filepath.Join(gw.args.src, name)
		names  []string
		id     string
		post   *Post
		msrc   string
		ok     bool
		links  map[string]string
		lnames []string
		fmap   template.FuncMap
	)
	if names, err = gw.readDir(src); err != nil {
		gw.log.Printf("Posts directory not found %v\n", src)
		// Fail silently
		return nil
	}
	links = make(map[string]string)
	for _, id = range names {
		msrc = filepath.Join(name, id, "meta.yaml")
		if post, ok = gw.site.Posts[id]; ok == false {
			post = &Post{
				Id:     id,
				SrcDir: filepath.Join(src, id),
				site:   gw.site,
			}
			gw.site.Posts[id] = post
		}
		if post.meta, err = gw.parsePostMeta(msrc); err != nil {
			return
		}
		if lnames, err = gw.readDir(filepath.Join(src, id)); err != nil {
			return
		}
		var p string
		if p, err = post.Path(); err != nil {
			return
		}
		links[id] = p
		for _, l := range lnames {
			links[filepath.Join(id, l)] = filepath.Join(p, l)
		}
	}
	fmap = template.FuncMap{
		"link": func(i string) string {
			return links[i]
		},
	}
	for id, post = range gw.site.Posts {
		if err = gw.renderPost(post, &fmap); err != nil {
			return err
		}
	}
	return
}

// Parses general site configuration from the source directory.
func (gw *GhostWriter) parseSiteMeta() (err error) {
	src := filepath.Join(gw.args.src, gw.args.config)
	gw.log.Printf("Parsing site meta %v\n", src)
	gw.site.meta = &SiteMeta{}
	return gw.unyaml(src, gw.site.meta)
}

// Parses root templates from the given template path.
func (gw *GhostWriter) parseTemplates() (err error) {
	var (
		src   = filepath.Join(gw.args.src, gw.args.templates)
		names []string
		id    string
		text  string
	)
	gw.templates = template.New("root")
	if names, err = gw.readDir(src); err != nil {
		gw.log.Printf("Templates directory not found %v\n", src)
		// Fail silently
		return nil
	}
	for _, n := range names {
		if text, err = gw.readFile(filepath.Join(src, n)); err != nil {
			return
		}
		id = strings.Replace(n, filepath.Ext(n), "", -1)
		if _, err = gw.templates.New(id).Parse(text); err != nil {
			return
		}
		gw.log.Printf("Created template with name %v\n", id)
	}
	return nil
}

// Reads directory contents from the given path and returns file names.
func (gw *GhostWriter) readDir(path string) (names []string, err error) {
	var f fauxfile.File
	if f, err = gw.fs.Open(path); err != nil {
		return
	}
	defer f.Close()
	names, err = f.Readdirnames(-1)
	return
}

// Reads a file from the given path and returns a string of the contents.
func (gw *GhostWriter) readFile(path string) (out string, err error) {
	var (
		f   fauxfile.File
		fi  os.FileInfo
		buf []byte
	)
	if f, err = gw.fs.Open(path); err != nil {
		return
	}
	defer f.Close()
	if fi, err = f.Stat(); err != nil {
		return
	}
	buf = make([]byte, fi.Size())
	if _, err = f.Read(buf); err != nil {
		if err != io.EOF {
			return
		}
		err = nil
	}
	out = string(buf)
	return
}

// Renders miscellaneous files, including static content, into output dir.
// Returns a non-nil error if something went wrong.
func (gw *GhostWriter) renderMisc() (err error) {
	var (
		name  = gw.args.static
		queue []string
		names []string
		p     string
		n     string
		src   string
		dst   string
		x     int
		i     os.FileInfo
	)
	if queue, err = gw.readDir(gw.args.src); err != nil {
		return
	}
	for len(queue) > 0 {
		p = queue[0]
		queue = queue[1:]
		switch p {
		case gw.args.posts:
			continue
		case gw.args.templates:
			continue
		}
		src = filepath.Join(gw.args.src, p)
		dst = filepath.Join(gw.args.dst, p)
		if i, err = gw.fs.Stat(src); err != nil {
			// Passed in path
			if name == p {
				gw.log.Printf("Static dir not found %v\n", src)
				// Fail silently
				return nil
			}
			return
		}
		if i.IsDir() {
			if names, err = gw.readDir(src); err != nil {
				return
			}
			for x, n = range names {
				names[x] = filepath.Join(p, n)
			}
			queue = append(queue, names...)
			gw.log.Printf("Creating %v\n", dst)
			if err = gw.fs.Mkdir(dst, i.Mode()); err != nil {
				str := err.(*os.PathError).Err.Error()
				if str == "file exists" {
					// Don't fail if the directory exists.
					err = nil
					continue
				}
				gw.log.Printf("Problem creating %v\n", dst)
				return
			}
		} else {
			switch filepath.Ext(src) {
			case ".tmpl":
				dst = fmt.Sprintf("%v.html", dst[:len(dst)-5])
				gw.log.Printf("Rendering %v to %v\n", src, dst)
				if err = gw.renderTemplate(src, dst); err != nil {
					return
				}
			default:
				gw.log.Printf("Copying %v to %v\n", src, dst)
				if _, err = gw.copyFile(src, dst); err != nil {
					return
				}
			}
		}
	}
	return
}

// Renders the initalized Post object into an HTML file in the destination.
func (gw *GhostWriter) renderPost(post *Post, fmap *template.FuncMap) (err error) {
	var (
		fdst     fauxfile.File
		src      string
		dst      string
		postpath string
		postbody string
		body     *bytes.Buffer
		mdbody   *bytes.Buffer
		writer   *bufio.Writer
		parser   *markdown.Parser
		tmpl     *template.Template
		names    []string
	)
	if postpath, err = post.Path(); err != nil {
		return
	}
	src = filepath.Join(post.SrcDir, "body.md")
	dst = path.Join(gw.args.dst, postpath, "index.html")
	if postbody, err = gw.readFile(src); err != nil {
		return
	}
	gw.fs.MkdirAll(path.Dir(dst), 0755)
	if fdst, err = gw.fs.Create(dst); err != nil {
		return
	}
	defer fdst.Close()
	if names, err = gw.readDir(post.SrcDir); err != nil {
		return
	}
	for _, name := range names {
		switch filepath.Ext(name) {
		case ".md":
		case ".yaml":
		default:
			// Copy other files into destination-they're content.
			s := filepath.Join(post.SrcDir, name)
			d := filepath.Join(gw.args.dst, postpath, name)
			gw.copyFile(s, d)
		}
	}

	// Render post body against links map.
	tmpl, err = template.New("body").Funcs(*fmap).Parse(postbody)
	if err != nil {
		return
	}
	body = new(bytes.Buffer)
	if err = tmpl.Execute(body, nil); err != nil {
		return
	}

	// Render markdown
	mdbody = new(bytes.Buffer)
	parser = markdown.NewParser(&markdown.Extensions{Smart: true})
	parser.Markdown(body, markdown.ToHTML(mdbody))
	post.Body = mdbody.String()

	// Render post into site template.
	writer = bufio.NewWriter(fdst)
	data := map[string]interface{}{
		"Post": post,
		"Site": gw.site,
	}
	// Should render "post" by default.
	err = gw.templates.Lookup("global").Execute(writer, data)
	writer.Flush()
	return
}

// Renders a Go template from the given path to the output path.
func (gw *GhostWriter) renderTemplate(src string, dst string) (err error) {
	var (
		text   string
		clone  *template.Template
		tmpl   *template.Template
		writer *bufio.Writer
		f      fauxfile.File
	)
	if text, err = gw.readFile(src); err != nil {
		return
	}
	if tmpl, err = template.New(src).Parse(text); err != nil {
		return
	}
	if clone, err = gw.mergeTemplate(tmpl); err != nil {
		return
	}
	if f, err = gw.fs.Create(dst); err != nil {
		return
	}
	writer = bufio.NewWriter(f)
	err = clone.ExecuteTemplate(writer, "global", gw.site)
	writer.Flush()
	return
}

// Deserializes the yaml file at the given path to the supplied object.
func (gw *GhostWriter) unyaml(path string, out interface{}) (err error) {
	var (
		file fauxfile.File
		info os.FileInfo
		data []byte
	)
	if file, err = gw.fs.Open(path); err != nil {
		return
	}
	defer file.Close()
	if info, err = file.Stat(); err != nil {
		return
	}
	data = make([]byte, info.Size())
	if _, err = file.Read(data); err != nil {
		return
	}
	err = goyaml.Unmarshal(data, out)
	return
}

// Represents a post for templating purposes.
type Post struct {
	Id     string
	Body   string
	SrcDir string
	meta   *PostMeta
	site   *Site
}

// Returns the date of the post, as configured in the post metadata.
func (p *Post) Date() (t time.Time, err error) {
	return time.Parse(p.site.meta.DateFormat, p.meta.Date)
}

// Returns the date of the post in the configured path format.
func (p *Post) DatePath() (s string) {
	var t time.Time
	t, _ = p.Date() // T should zero value if error
	return t.Format(p.site.meta.DateFormat)
}

// Returns the URL-friendly identifier for the post.
func (p *Post) Slug() (s string) {
	s = strings.ToLower(p.meta.Slug)
	return
}

// Returns the human-friendly title of the post.
func (p *Post) Title() (s string) {
	s = p.meta.Title
	return
}

// Returns the relative URL path for the post.
func (p *Post) Path() (out string, err error) {
	var (
		t *template.Template
		b *bytes.Buffer
	)
	if t, err = p.site.PathTemplate(); err != nil {
		return
	}
	b = bytes.NewBufferString("")
	if err = t.Execute(b, p); err != nil {
		return
	}
	out = b.String()
	return
}

// Returns the fully-qualified link for the post.
func (p *Post) Permalink() (s string) {
	path, _ := p.Path()
	s = fmt.Sprintf("%v%v", p.site.Root(), path)
	return
}

// Returns the fully-qualified link for the post as a url.URL object.
func (p *Post) URL() (u *url.URL, err error) {
	var postpath string
	if postpath, err = p.Path(); err != nil {
		return
	}
	return url.Parse(path.Join(p.site.meta.Root, postpath))
}

// Serializable metadata about the post.
type PostMeta struct {
	Tags  []string
	Title string
	Date  string
	Slug  string
}

// Represents the site for templating purposes.
type Site struct {
	Posts        map[string]*Post
	meta         *SiteMeta
	pathTemplate *template.Template
}

// Returns the title of the site.
func (s *Site) Title() string {
	return s.meta.Title
}

// Returns the root of the site's URL.
func (s *Site) Root() string {
	return s.meta.Root
}

// Returns a template suitable for rendering post URLs.
func (s *Site) PathTemplate() (t *template.Template, err error) {
	if s.pathTemplate == nil {
		s.pathTemplate, err = template.New("path").Parse(s.meta.PathFormat)
		if err != nil {
			return
		}
	}
	t = s.pathTemplate
	return
}

// Serializable metadata about the site.
type SiteMeta struct {
	Title      string
	Root       string
	PathFormat string
	DateFormat string
}

// Main routine.
func main() {
	a := DefaultArgs()
	flag.StringVar(&a.src, "src", "src", "Path to src files.")
	flag.StringVar(&a.dst, "dst", "dst", "Build output directory.")
	flag.Parse()
	w := NewGhostWriter(&fauxfile.RealFilesystem{}, a)
	if err := w.Process(); err != nil {
		fmt.Println(err)
	}
}
