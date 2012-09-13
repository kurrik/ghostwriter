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

type Args struct {
	src string
	dst string
}

type Configuration struct {
	fs fauxfile.Filesystem
}

type GhostWriter struct {
	args      *Args
	fs        fauxfile.Filesystem
	log       *log.Logger
	site      *Site
	templates map[string]*template.Template
}

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

func (gw *GhostWriter) Process() (err error) {
	if err = gw.parseSiteMeta("config.yaml"); err != nil {
		return
	}
	if err = gw.copyStatic("static"); err != nil {
		return
	}
	if err = gw.parseTemplates("templates"); err != nil {
		return
	}
	if err = gw.parsePosts("posts"); err != nil {
		return
	}
	return
}

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

func (gw *GhostWriter) copyStatic(name string) (err error) {
	var (
		queue []string
		names []string
		p     string
		n     string
		src   string
		dst   string
		x     int
		i     os.FileInfo
	)
	queue = []string{name}
	for len(queue) > 0 {
		p = queue[0]
		queue = queue[1:]
		src = filepath.Join(gw.args.src, p)
		dst = filepath.Join(gw.args.dst, p)
		if i, err = gw.fs.Stat(src); err != nil {
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
				gw.log.Printf("Problem creating %v\n", dst)
				return
			}
		} else {
			gw.log.Printf("Copying %v to %v\n", src, dst)
			if _, err = gw.copyFile(src, dst); err != nil {
				return
			}
		}
	}
	return
}

func (gw *GhostWriter) parsePostMeta(path string) (meta *PostMeta, err error) {
	src := filepath.Join(gw.args.src, path)
	gw.log.Printf("Parsing site meta %v\n", src)
	meta = &PostMeta{}
	err = gw.unyaml(src, meta)
	return
}

func (gw *GhostWriter) parsePosts(name string) (err error) {
	var (
		src   = filepath.Join(gw.args.src, name)
		names []string
		id    string
		post  *Post
		msrc  string
		ok    bool
	)
	if names, err = gw.readDir(src); err != nil {
		gw.log.Printf("Posts directory not found %v\n", src)
		// Fail silently
		return nil
	}
	for _, id = range names {
		msrc = filepath.Join(name, id, "meta.yaml")
		if post, ok = gw.site.Posts[id]; ok == false {
			post = &Post{
				Id:   id,
				site: gw.site,
			}
			gw.site.Posts[id] = post
		}
		if post.meta, err = gw.parsePostMeta(msrc); err != nil {
			return
		}
	}
	for id, post = range gw.site.Posts {
		if err = gw.renderPost(post, src); err != nil {
			return err
		}
	}
	return
}

func (gw *GhostWriter) parseSiteMeta(path string) (err error) {
	src := filepath.Join(gw.args.src, path)
	gw.log.Printf("Parsing site meta %v\n", src)
	gw.site.meta = &SiteMeta{}
	return gw.unyaml(src, gw.site.meta)
}

func (gw *GhostWriter) parseTemplates(path string) (err error) {
	var (
		src   = filepath.Join(gw.args.src, path)
		names []string
		id    string
		text  string
		tmpl  *template.Template
	)
	gw.templates = make(map[string]*template.Template)
	if names, err = gw.readDir(src); err != nil {
		gw.log.Printf("Templates directory not found %v\n", src)
		// Fail silently
		return nil
	}
	for _, n := range names {
		if text, err = gw.readFile(filepath.Join(src, n)); err != nil {
			return
		}
		if tmpl, err = template.New(id).Parse(text); err != nil {
			return
		}
		id = strings.Replace(n, filepath.Ext(n), "", -1)
		gw.templates[id] = tmpl
	}
	return nil
}

func (gw *GhostWriter) readDir(path string) (names []string, err error) {
	var f fauxfile.File
	if f, err = gw.fs.Open(path); err != nil {
		return
	}
	defer f.Close()
	names, err = f.Readdirnames(-1)
	return
}

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

func (gw *GhostWriter) renderPost(post *Post, srcdir string) (err error) {
	var (
		fsrc     fauxfile.File
		fdst     fauxfile.File
		src      string
		dst      string
		postpath string
		body     *bytes.Buffer
		writer   *bufio.Writer
		parser   *markdown.Parser
	)
	if postpath, err = post.Path(); err != nil {
		return
	}
	src = filepath.Join(srcdir, post.Id, "body.md")
	dst = path.Join(gw.args.dst, postpath)
	if fsrc, err = gw.fs.Open(src); err != nil {
		return
	}
	defer fsrc.Close()
	gw.fs.MkdirAll(path.Dir(dst), 0755)
	if fdst, err = gw.fs.Create(dst); err != nil {
		return
	}
	defer fdst.Close()
	body = new(bytes.Buffer)
	parser = markdown.NewParser(&markdown.Extensions{Smart: true})
	parser.Markdown(fsrc, markdown.ToHTML(body))
	post.Body = body.String()
	writer = bufio.NewWriter(fdst)
	data := map[string]interface{}{
		"Post": post,
		"Site": gw.site,
	}
	err = gw.templates["post"].Execute(writer, data)
	writer.Flush()
	return
}

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

type Post struct {
	Id   string
	Body string
	meta *PostMeta
	site *Site
}

func (p *Post) Date() (t time.Time, err error) {
	return time.Parse(p.site.meta.DateFormat, p.meta.Date)
}

func (p *Post) DatePath() (s string) {
	var t time.Time
	t, _ = p.Date() // T should zero value if error
	return t.Format(p.site.meta.DateFormat)
}

func (p *Post) Slug() (s string) {
	s = strings.ToLower(p.meta.Slug)
	return
}

func (p *Post) Title() (s string) {
	s = p.meta.Title
	return
}

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

func (p *Post) Permalink() (s string) {
	path, _ := p.Path()
	s = fmt.Sprintf("%v%v", p.site.Root(), path)
	return
}

func (p *Post) URL() (u *url.URL, err error) {
	var postpath string
	if postpath, err = p.Path(); err != nil {
		return
	}
	return url.Parse(path.Join(p.site.meta.Root, postpath))
}

type PostMeta struct {
	Tags  []string
	Title string
	Date  string
	Slug  string
}

type Site struct {
	Posts        map[string]*Post
	meta         *SiteMeta
	pathTemplate *template.Template
}

func (s *Site) Title() string {
	return s.meta.Title
}

func (s *Site) Root() string {
	return s.meta.Root
}

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

type SiteMeta struct {
	Title      string
	Root       string
	PathFormat string
	DateFormat string
}

func main() {
	a := &Args{}
	flag.StringVar(&a.src, "src", "src", "Path to src files.")
	flag.StringVar(&a.dst, "dst", "dst", "Build output directory.")
	flag.Parse()
	w := NewGhostWriter(&fauxfile.RealFilesystem{}, a)
	if err := w.Process(); err != nil {
		fmt.Println(err)
	}
}
