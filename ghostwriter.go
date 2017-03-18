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
	"github.com/kurrik/tmpl"
	"github.com/russross/blackfriday"
	"gopkg.in/yaml.v2"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
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
	rootTemplate *tmpl.Templates
	postTemplate string
	tagsTemplate string
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
	if gw.args.before != "" {
		var (
			cmd *exec.Cmd
			out bytes.Buffer
		)
		cmd = exec.Command(gw.args.before)
		cmd.Stdout = &out
		gw.log.Printf("Running %v\n", gw.args.before)
		if err = cmd.Run(); err != nil {
			return
		}
		gw.log.Printf("Output:\n%v\n", out.String())
	}
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

	defer func() {
		if err := fsrc.Close(); err != nil {
			gw.log.Printf("Problem closing %v: %v\n", src, err)
		}
	}()

	if fdst, err = gw.fs.Create(dst); err != nil {
		return
	}

	defer func() {
		if err := fdst.Close(); err != nil {
			gw.log.Printf("Problem closing %v: %v\n", dst, err)
		}
	}()

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
		path      string
		names     []string
		id        string
		text      string
		foundPost bool = false
		foundRoot bool = false
		foundTags bool = false
	)
	gw.rootTemplate = tmpl.NewTemplates()
	gw.rootTemplate.SetFilesystem(gw.fs)
	if names, err = gw.readDir(src); err != nil {
		gw.log.Printf("Templates directory not found %v\n", src)
		// Fail silently
		return nil
	}
	for _, n := range names {
		path = filepath.Join(src, n)
		id = strings.Replace(n, filepath.Ext(n), "", -1)
		if n == gw.args.postTemplate {
			foundPost = true
			if text, err = gw.readFile(path); err != nil {
				return
			}
			gw.postTemplate = text
			gw.log.Printf("Found post template with name %v\n", id)
		} else if n == gw.args.tagsTemplate {
			foundTags = true
			if text, err = gw.readFile(path); err != nil {
				return
			}
			gw.tagsTemplate = text
			gw.log.Printf("Found tags template with name %v\n", id)
		} else {
			if err = gw.rootTemplate.AddTemplateFromFile(path); err != nil {
				return
			}
			foundRoot = true
			gw.log.Printf("Found root template with name %v\n", id)
		}
	}
	if foundRoot == false {
		gw.log.Printf("No root template found.")
		if foundPost {
			// Not sure if this makes the greatest sense, but use
			// the post template as the base template.  Maybe they
			// just put all the HTML in there?
			gw.rootTemplate.AddTemplate(gw.postTemplate)
		} else {
			gw.rootTemplate.AddTemplate("")
		}
	}
	if foundPost == false {
		err = fmt.Errorf("No post template at: %v", gw.args.postTemplate)
	}
	if foundTags == false {
		// Not an error
	}
	//fmt.Printf("TEMPLATES: %v\n", gw.rootTemplate.Templates())
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

// Returns a base set of functions for use in templates.
func (gw *GhostWriter) getFuncMap() *template.FuncMap {
	return &template.FuncMap{
		"timeformat": func(t time.Time, f string) string {
			return t.Format(f)
		},
		"textcontent": func(s string) string {
			rex, _ := regexp.Compile("<[^>]*>")
			return rex.ReplaceAllLiteralString(s, "")
		},
	}
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
		str      string
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
	for i := 0; i < len(names); i++ {
		var (
			name     string
			namePath string
			nameInfo os.FileInfo
			subNames []string
			subPath  string
		)
		name = names[i]
		namePath = filepath.Join(post.SrcDir, name)
		if nameInfo, err = gw.fs.Stat(namePath); err != nil {
			return
		}
		if nameInfo.IsDir() {
			if subNames, err = gw.readDir(namePath); err != nil {
				return
			}
			subPath = filepath.Join(gw.args.dst, postpath, name)
			gw.fs.MkdirAll(subPath, 0755)
			for _, subName := range subNames {
				names = append(names, filepath.Join(name, subName))
			}
			continue
		}
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

	fmap = gw.getFuncMap()
	(*fmap)["link"] = func(i string) string {
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
	}
	(*fmap)["include"] = func(i string) (contents string) {
		var (
			includeErr error
			fixedPath  string
		)
		fixedPath = filepath.Join(post.SrcDir, i)
		if contents, includeErr = gw.readFile(fixedPath); includeErr != nil {
			contents = fmt.Sprintf("[[ERROR: Could not read %v]]", fixedPath)
		}
		return
	}
	(*fmap)["slice"] = func(data ...interface{}) (out []interface{}) {
		return data
	}
	(*fmap)["map"] = func(data ...interface{}) (out map[string]interface{}) {
		var key string
		out = map[string]interface{}{}
		for i, datum := range data {
			if i%2 == 0 {
				key = datum.(string)
			} else {
				out[key] = datum
			}
		}
		return
	}
	(*fmap)["imagemeta"] = func(path string) (img ImageMeta, ferr error) {
		var (
			srcPath string = filepath.Join(post.SrcDir, path)
			dstPath string = filepath.Join(postpath, path)
		)
		if img, ferr = NewImageMeta(gw.fs, srcPath, dstPath); ferr != nil {
			ferr = fmt.Errorf("Could not load image metadata: %v", ferr)
			return
		}
		return
	}

	if len(postbody) > 0 {
		// Render post body against function declarations
		if tmpl, err = gw.rootTemplate.MergeInto(template.New("body")); err != nil {
			return
		}
		if tmpl, err = tmpl.Lookup("body").Funcs(*fmap).Parse(postbody); err != nil {
			return
		}
		body = new(bytes.Buffer)
		if err = tmpl.Lookup("body").Execute(body, nil); err != nil {
			return
		}

		// Render markdown
		post.Body = string(blackfriday.MarkdownCommon(body.Bytes()))

		// Check for snippet
		if index = strings.Index(post.Body, "<!--BREAK-->"); index != -1 {
			post.Snippet = post.Body[0:index]
		}
	}

	// Render post into site template.
	writer = bufio.NewWriter(fdst)
	data := map[string]interface{}{
		"Post": post,
		"Site": gw.site,
	}
	if str, err = gw.rootTemplate.RenderText(gw.postTemplate, data); err != nil {
		return
	}
	writer.Write([]byte(str))
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
		writer  *bufio.Writer
		fdst    fauxfile.File
		str     string
	)
	if gw.tagsTemplate == "" {
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
		if str, err = gw.rootTemplate.RenderText(gw.tagsTemplate, data); err != nil {
			return
		}
		writer.Write([]byte(str))
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
		writer *bufio.Writer
		f      fauxfile.File
		data   map[string]interface{}
		str    string
	)
	if f, err = gw.fs.Create(dst); err != nil {
		return
	}
	writer = bufio.NewWriter(f)
	data = map[string]interface{}{
		"Site": gw.site,
	}
	if str, err = gw.rootTemplate.RenderFile(src, data); err != nil {
		return
	}
	writer.Write([]byte(str))
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
	err = yaml.Unmarshal(data, out)
	return
}

// Serializable metadata about the post.
type PostMeta struct {
	Tags    []string
	Title   string
	Date    string
	Slug    string
	Scripts []string
	Styles  []string
}

// Represents a tag and the number of posts with that tag.
type TagCount struct {
	Tag   string
	Count int
}

// A list of TagCounts
type TagCounts []*TagCount

// Compares two posts.
func (tc TagCounts) Less(i int, j int) bool {
	ci := tc[i].Count
	cj := tc[j].Count
	return ci > cj
}

// Returns the length of a set of tag counts.
func (tc TagCounts) Len() int {
	return len(tc)
}

// Swaps two tag counts in the given positions.
func (tc TagCounts) Swap(i int, j int) {
	tc[i], tc[j] = tc[j], tc[i]
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
