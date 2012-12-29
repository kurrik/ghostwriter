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
	"fmt"
	"github.com/kurrik/fauxfile"
	"github.com/russross/blackfriday"
	"io"
	"launchpad.net/goyaml"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

// Master Control Program.
type GhostWriter struct {
	args         *Args
	fs           fauxfile.Filesystem
	log          *log.Logger
	site         *Site
	links        map[string]string
	rootTemplate *template.Template
	postTemplate *template.Template
	tagsTemplate *template.Template
}

// Creates a new GhostWriter.
func NewGhostWriter(fs fauxfile.Filesystem, args *Args) *GhostWriter {
	gw := &GhostWriter{
		args:  args,
		fs:    fs,
		log:   log.New(os.Stderr, "", log.LstdFlags),
		links: make(map[string]string),
		site: &Site{
			Posts:    make(map[string]*Post),
			Tags:     make(map[string]Posts),
			Rendered: time.Now(),
		},
	}
	return gw
}

// Parses the src directory, rendering into dst as needed.
func (gw *GhostWriter) Process() (err error) {
	gw.links = make(map[string]string)
	gw.site = &Site{
		Posts:    make(map[string]*Post),
		Tags:     make(map[string]Posts),
		Rendered: time.Now(),
	}
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
	if err = gw.renderPosts(); err != nil {
		return
	}
	if err = gw.renderTags(); err != nil {
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

// Returns true if the specified path is a directory.
func (gw *GhostWriter) isDir(path string) bool {
	var (
		info os.FileInfo
		err  error
	)
	if info, err = gw.fs.Stat(path); err != nil {
		return false
	}
	return info.IsDir()
}

// Returns a copy of the global template with the supplied template merged in.
func (gw *GhostWriter) mergeTemplate(t *template.Template) (out *template.Template, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Seems to be a bug with cloning empty templates.
			err = fmt.Errorf("Problem cloning template: %v", r)
		}
	}()
	if out, err = gw.rootTemplate.Clone(); err != nil {
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
	gw.log.Printf("Parsing post meta %v\n", src)
	meta = &PostMeta{}
	if err = gw.unyaml(src, meta); err != nil {
		return
	}
	if meta.Date == "" {
		err = fmt.Errorf("Post meta must include date")
		return
	}
	if meta.Slug == "" {
		err = fmt.Errorf("Post meta must include slug")
		return
	}
	if meta.Title == "" {
		err = fmt.Errorf("Post meta must include title")
		return
	}
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
		lnames []string
	)
	if names, err = gw.readDir(src); err != nil {
		gw.log.Printf("Posts directory not found %v\n", src)
		// Fail silently
		return nil
	}
	for _, id = range names {
		if !gw.isDir(filepath.Join(src, id)) {
			continue
		}
		msrc = filepath.Join(name, id, "meta.yaml")
		if post, ok = gw.site.Posts[id]; ok == false {
			post = &Post{
				Id:     id,
				SrcDir: filepath.Join(src, id),
				site:   gw.site,
			}
		}
		if post.meta, err = gw.parsePostMeta(msrc); err != nil {
			// Not a post, but don't raise an error.
			gw.log.Printf("Invalid post at %v: %v\n", msrc, err)
			return nil
		}
		// Add to site posts after determining whether it's a real post.
		gw.site.Posts[id] = post
		if lnames, err = gw.readDir(filepath.Join(src, id)); err != nil {
			return
		}
		var p string
		if p, err = post.Path(); err != nil {
			return
		}
		gw.links[id] = p
		for _, l := range lnames {
			gw.links[filepath.Join(id, l)] = filepath.Join(p, l)
		}
		for _, tag := range post.Tags() {
			gw.site.Tags[tag] = append(gw.site.Tags[tag], post)
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
		src       string = filepath.Join(gw.args.src, gw.args.templates)
		names     []string
		id        string
		text      string
		foundPost bool = false
		foundRoot bool = false
		foundTags bool = false
	)
	gw.rootTemplate = template.New("root")
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
		if n == gw.args.postTemplate {
			foundPost = true
			gw.postTemplate = template.New("post")
			_, err = gw.postTemplate.Parse(text)
			gw.log.Printf("Parsed post template with name %v\n", id)
		} else if n == gw.args.tagsTemplate {
			foundTags = true
			gw.tagsTemplate = template.New("tags")
			_, err = gw.tagsTemplate.Parse(text)
			gw.log.Printf("Parsed tags template with name %v\n", id)
		} else {
			foundRoot = true
			_, err = gw.rootTemplate.Parse(text)
			gw.log.Printf("Parsed root template with name %v\n", id)
		}
		if err != nil {
			return
		}
	}
	if foundRoot == false {
		gw.log.Printf("No root template found.")
		if foundPost {
			// Not sure if this makes the greatest sense, but use
			// the post template as the base template.  Maybe they
			// just put all the HTML in there?
			gw.rootTemplate = gw.postTemplate
		} else {
			gw.rootTemplate.Parse("")
		}
	}
	if foundPost == false {
		err = fmt.Errorf("No post template at: %v", gw.args.postTemplate)
	}
	if foundTags == false {
		// Not an error
	}
	return
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
				dst = dst[:len(dst)-5]
				if filepath.Ext(dst) == "" {
					dst = fmt.Sprintf("%v.html", dst)
				}
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

// Renders all of the posts in the site.
func (gw *GhostWriter) renderPosts() (err error) {
	var (
		post *Post
	)
	for _, post = range gw.site.Posts {
		if err = gw.renderPost(post); err != nil {
			return
		}
	}
	return
}

// Renders the initalized Post object into an HTML file in the destination.
func (gw *GhostWriter) renderPost(post *Post) (err error) {
	var (
		fdst     fauxfile.File
		src      string
		dst      string
		postpath string
		postbody string
		body     *bytes.Buffer
		writer   *bufio.Writer
		tmpl     *template.Template
		names    []string
		fmap     *template.FuncMap
		index    int
	)
	if postpath, err = post.Path(); err != nil {
		return
	}
	src = filepath.Join(post.SrcDir, "body.md")
	dst = path.Join(gw.args.dst, postpath, "index.html")
	if postbody, err = gw.readFile(src); err != nil {
		// A missing body is not an error, just assume a blank entry.
		postbody = ""
		err = nil
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

	fmap = &template.FuncMap{
		"link": func(i string) string {
			var (
				locali string
				link   string
				ok     bool
			)
			locali = fmt.Sprintf("%v/%v", post.Id, i)
			if link, ok = gw.links[locali]; ok {
				return link
			}
			return gw.links[i]
		},
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
	post.Body = string(blackfriday.MarkdownCommon(body.Bytes()))

	// Check for snippet
	if index = strings.Index(post.Body, "<!--BREAK-->"); index != -1 {
		post.Snippet = post.Body[0:index]
	}

	// Render post into site template.
	writer = bufio.NewWriter(fdst)
	data := map[string]interface{}{
		"Post": post,
		"Site": gw.site,
	}
	if tmpl, err = gw.mergeTemplate(gw.postTemplate); err != nil {
		return
	}
	err = tmpl.Execute(writer, data)
	writer.Flush()
	return
}

// Renders all of the posts in the site.
func (gw *GhostWriter) renderTags() (err error) {
	var (
		posts   Posts
		tag     string
		dst     string
		tagpath string
		tmpl    *template.Template
		writer  *bufio.Writer
		fdst    fauxfile.File
	)
	if gw.tagsTemplate == nil {
		return
	}
	if tmpl, err = gw.mergeTemplate(gw.tagsTemplate); err != nil {
		return
	}
	for tag, posts = range gw.site.Tags {
		tagpath = gw.site.TagPath(tag)
		dst = path.Join(gw.args.dst, tagpath, "index.html")
		gw.fs.MkdirAll(path.Dir(dst), 0755)
		if fdst, err = gw.fs.Create(dst); err != nil {
			return
		}
		defer fdst.Close()
		writer = bufio.NewWriter(fdst)
		sort.Sort(ByDateDesc{posts})
		data := map[string]interface{}{
			"Tag":   tag,
			"Posts": posts,
			"Site":  gw.site,
		}
		err = tmpl.Execute(writer, data)
		writer.Flush()
		if err != nil {
			return
		}
		fdst.Close()
	}
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
		data   map[string]interface{}
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
	data = map[string]interface{}{
		"Site": gw.site,
	}
	err = clone.Execute(writer, data)
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
	Id      string
	Body    string
	Snippet string
	SrcDir  string
	meta    *PostMeta
	site    *Site
}

// Returns the date of the post, as configured in the post metadata.
func (p *Post) Date() (t time.Time, err error) {
	return time.Parse(p.site.meta.DateFormat, p.meta.Date)
}

// Returns the post date or panics if it has an error.
func (p *Post) SureDate() (t time.Time) {
	var err error
	if t, err = p.Date(); err != nil {
		panic(fmt.Sprintf("Could not get date from post %v", p.Id))
	}
	return
}

// Returns the date of the post in the configured path format.
func (p *Post) DatePath() (s string) {
	var t time.Time
	t, _ = p.Date() // T should zero value if error
	return t.Format(p.site.meta.DateFormat)
}

// Returns the human-friendly date of this post.
func (p *Post) FormattedDate() (s string) {
	s = p.SureDate().Format("Mon Jan _2, 2006")
	return
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

// Returns the names of the tags this post belongs to.
func (p *Post) Tags() (t []string) {
	t = p.meta.Tags
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

// Returns the next post, chronologically.
func (p *Post) Next() *Post {
	return p.site.NextPost(p)
}

// Returns the previous post, chronologically.
func (p *Post) Prev() *Post {
	return p.site.PrevPost(p)
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
	tagsTemplate *template.Template
	Tags         map[string]Posts
	Rendered     time.Time
}

// Returns the path for a given tag
func (s *Site) TagPath(tag string) string {
	var (
		err error
		b   *bytes.Buffer
		d   map[string]interface{}
	)
	if s.tagsTemplate == nil {
		s.tagsTemplate, err = template.New("tags").Parse(s.meta.TagsFormat)
		if err != nil {
			panic("Could not parse tags format")
		}
	}
	b = bytes.NewBufferString("")
	d = map[string]interface{}{
		"Tag": tag,
	}
	if err = s.tagsTemplate.Execute(b, d); err != nil {
		panic(fmt.Sprintf("Could not get path for tag %v", tag))
	}
	return b.String()
}

// Returns the title of the site.
func (s *Site) Title() string {
	return s.meta.Title
}

// Returns the root of the site's URL.
func (s *Site) Root() string {
	return s.meta.Root
}

// Returns the site author.
func (s *Site) Author() string {
	return s.meta.Author
}

// Returns the site email address.
func (s *Site) Email() string {
	return s.meta.Email
}

// Returns the posts of the site in desending chronological order.
func (s *Site) PostsByDate() Posts {
	p := PostsFromMap(s.Posts)
	sort.Sort(ByDateDesc{p})
	return p
}

// Returns the first N of the posts by date.
func (s *Site) RecentPosts() Posts {
	p := s.PostsByDate()
	lim := len(p)
	if s.meta.RecentCount < lim {
		lim = s.meta.RecentCount
	}
	return s.PostsByDate()[0:lim]
}

// Returns the index of the given post in the given list of posts
func (s *Site) postIndex(posts Posts, p *Post) int {
	if p == nil {
		return -1
	}
	for i, post := range posts {
		if post == p {
			return i
		}
	}
	return -1
}

// Returns the next post chronologically given a reference post.
func (s *Site) NextPost(p *Post) *Post {
	posts := s.PostsByDate()
	i := s.postIndex(posts, p)
	if i > 0 {
		return posts[i-1]
	}
	return nil
}

// Returns the previous post chronologically given a reference post.
func (s *Site) PrevPost(p *Post) *Post {
	posts := s.PostsByDate()
	i := s.postIndex(posts, p)
	if i != -1 && i < len(posts)-1 {
		return posts[i+1]
	}
	return nil
}

// A list of posts.
type Posts []*Post

// Returns the length of the list.
func (p Posts) Len() int {
	return len(p)
}

// Swaps two posts in the given positions.
func (p Posts) Swap(i int, j int) {
	p[i], p[j] = p[j], p[i]
}

// Given a map of id => *Post, return a list in arbitrary order.
func PostsFromMap(m map[string]*Post) Posts {
	p := make(Posts, len(m))
	i := 0
	for _, post := range m {
		p[i] = post
		i++
	}
	return p
}

// Wrapper for sorting posts chronologically, descending.
type ByDateDesc struct{ Posts }

// Compares two posts.
func (p ByDateDesc) Less(i int, j int) bool {
	di := p.Posts[i].SureDate()
	dj := p.Posts[j].SureDate()
	if di.Equal(dj) {
		return p.Posts[i].Id > p.Posts[j].Id
	}
	return di.After(dj)
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
	Title       string
	Root        string
	Author      string
	Email       string
	PathFormat  string
	DateFormat  string
	TagsFormat  string
	RecentCount int
}
