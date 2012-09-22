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
	"github.com/kurrik/fauxfile"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
)

func Setup() (gw *GhostWriter, fs *fauxfile.MockFilesystem) {
	fs = fauxfile.NewMockFilesystem()
	fs.MkdirAll("/home/test", 0755)
	fs.Chdir("/home/test")
	fs.Mkdir("src", 0755)
	args := DefaultArgs()
	args.dst = "build"
	gw = NewGhostWriter(fs, args)
	gw.log = log.New(ioutil.Discard, "", log.LstdFlags)
	return
}

func WriteFile(fs fauxfile.Filesystem, p string, data string) error {
	var (
		f   fauxfile.File
		err error
	)
	fs.MkdirAll(path.Dir(p), 0755)
	if f, err = fs.Create(p); err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write([]byte(data)); err != nil {
		return err
	}
	return nil
}

func ReadFile(fs fauxfile.Filesystem, path string) (data string, err error) {
	var (
		f  fauxfile.File
		fi os.FileInfo
	)
	if f, err = fs.Open(path); err != nil {
		return
	}
	defer f.Close()
	if fi, err = f.Stat(); err != nil {
		return
	}
	buf := make([]byte, fi.Size())
	if _, err = f.Read(buf); err != nil {
		if err != io.EOF {
			return
		}
		err = nil
	}
	data = string(buf)
	return
}

const SITE_META = `
title: Test blog
root: http://www.example.com
pathformat: /{{.DatePath}}/{{.Slug}}
dateformat: "2006-01-02"`

// Ensures that config files are parsed and values pulled out.
func TestParseSiteMeta(t *testing.T) {
	gw, fs := Setup()
	WriteFile(fs, "/home/test/src/config.yaml", SITE_META)
	if err := gw.parseSiteMeta(); err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}
	if gw.site.meta.Title != "Test blog" {
		t.Errorf("Bad title, got %v", gw.site.meta.Title)
	}
	if gw.site.meta.Root != "http://www.example.com" {
		t.Errorf("Bad root, got %v", gw.site.meta.Root)
	}
	if gw.site.meta.PathFormat != "/{{.DatePath}}/{{.Slug}}" {
		t.Errorf("Bad path format, got %v", gw.site.meta.PathFormat)
	}
	if gw.site.meta.DateFormat != "2006-01-02" {
		t.Errorf("Bad date format, got %v", gw.site.meta.DateFormat)
	}
}

const POST_META = `
date: 2012-09-07
slug: hello-world
title: Hello World!
tags:
  - hello
  - world`

// Ensures that post meta files are parsed and values pulled out.
func TestParsePostMeta(t *testing.T) {
	gw, fs := Setup()
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_META)
	meta, err := gw.parsePostMeta("posts/01-test/meta.yaml")
	if err != nil {
		t.Fatalf("parsePostMeta returned error: %v", err)
	}
	if meta.Title != "Hello World!" {
		t.Errorf("Bad title, got %v", meta.Title)
	}
	if meta.Date != "2012-09-07" {
		t.Errorf("Bad date, got %v", meta.Date)
	}
	if meta.Slug != "hello-world" {
		t.Errorf("Bad slug, got %v", meta.Slug)
	}
	if meta.Tags[0] != "hello" || meta.Tags[1] != "world" {
		t.Errorf("Bad tags, got %v", meta.Tags)
	}
}

// Ensures that static files are copied to the appropriate build locations.
func TestFilesCopiedToBuild(t *testing.T) {
	gw, fs := Setup()
	data1 := "javascript"
	data2 := "css"
	WriteFile(fs, "src/static/js/app.js", data1)
	WriteFile(fs, "src/static/css/app.css", data2)
	WriteFile(fs, "src/config.yaml", "")
	if err := gw.Process(); err != nil {
		t.Fatalf("%v", err)
	}
	if s, _ := ReadFile(fs, "build/static/js/app.js"); s != data1 {
		t.Errorf("Read: %v, Expected: %v", s, data1)
	}
	if s, _ := ReadFile(fs, "build/static/css/app.css"); s != data2 {
		t.Errorf("Read: %v, Expected: %v", s, data2)
	}
}

// Ensures post content is rendered appropriately.
func TestRenderContent(t *testing.T) {
	gw, fs := Setup()
	body := `
Hello World
===========
This is a fake post, for testing.

This is markdown
----------------`
	tmpl := `<!DOCTYPE html>
<html>
  <head>
    <title>{{.Site.Title}} - {{.Post.Title}}</title>
    <link rel="canonical" href="{{.Post.Permalink}}" />
  </head>
  <body>
{{template "body" .}}
  </body>
</html>`
	tmpl_post := `{{define "body"}}{{.Post.Body}}{{end}}`
	html := `<!DOCTYPE html>
<html>
  <head>
    <title>Test blog - Hello World!</title>
    <link rel="canonical" href="http://www.example.com/2012-09-07/hello-world" />
  </head>
  <body>
<h1>Hello World</h1>

<p>This is a fake post, for testing.</p>

<h2>This is markdown</h2>

  </body>
</html>`
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/global.tmpl", tmpl)
	WriteFile(fs, "src/templates/post.tmpl", tmpl_post)
	WriteFile(fs, "src/posts/01-test/body.md", body)
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_META)
	var (
		err error
		out string
	)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out, err = ReadFile(fs, "build/2012-09-07/hello-world/index.html"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != html {
		t.Fatalf("Read:\n%v\nExpected:\n%v", out, html)
	}
}

// Ensures post content (images, etc) are copied to build dir.
func TestPostContentCopied(t *testing.T) {
	var (
		err     error
		out     string
		content = "Content!"
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/global.tmpl", "")
	WriteFile(fs, "src/posts/01-test/body.md", "")
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_META)
	WriteFile(fs, "src/posts/01-test/content.png", content)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out, err = ReadFile(fs, "build/2012-09-07/hello-world/content.png"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != content {
		t.Fatalf("Read:\n%v\nExpected:\n%v", out, content)
	}
}

// Ensures links between posts are rendered
func TestRenderLinks(t *testing.T) {
	gw, fs := Setup()
	body1 := `
Post 1
======
This is a target post`
	meta1 := `
date: 2012-09-07
slug: hello-world`
	body2 := `
Post 2
======
This is a <a href="{{link "01-test"}}">link</a> to a post.
<img src="{{link "01-test/img.png"}}" />`
	meta2 := `
date: 2012-09-09
slug: hello-again`
	tmpl := `<html>{{.Post.Body}}</html>`
	html2 := `<html><h1>Post 2</h1>

<p>This is a <a href="/2012-09-07/hello-world">link</a> to a post.
<img src="/2012-09-07/hello-world/img.png" /></p>
</html>`
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/global.tmpl", tmpl)
	WriteFile(fs, "src/posts/01-test/body.md", body1)
	WriteFile(fs, "src/posts/01-test/img.png", "")
	WriteFile(fs, "src/posts/01-test/meta.yaml", meta1)
	WriteFile(fs, "src/posts/02-test/body.md", body2)
	WriteFile(fs, "src/posts/02-test/meta.yaml", meta2)
	var (
		err error
		out string
	)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out, err = ReadFile(fs, "build/2012-09-09/hello-again/index.html"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != html2 {
		t.Fatalf("Read:\n%v\nExpected:\n%v", out, html2)
	}
}

// Ensures the index page is rendered.
func TestRenderIndex(t *testing.T) {
	gw, fs := Setup()
	body1 := "Post 1"
	meta1 := "date: 2012-09-07\nslug: post1"
	body2 := "Post 2"
	meta2 := "date: 2012-09-08\nslug: post2"
	index := `{{define "body"}}
{{range .Posts}}
  <div>{{.Body}}</div>
{{end}}
{{end}}`
	tmpl := `<html>{{template "body" .}}</html>`
	post_tmpl := `{{define "body"}}{{.Post.Body}}{{end}}`
	html := `<html>

  <div><p>Post 1</p>
</div>

  <div><p>Post 2</p>
</div>

</html>`
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/global.tmpl", tmpl)
	WriteFile(fs, "src/templates/post.tmpl", post_tmpl)
	WriteFile(fs, "src/posts/01-test/body.md", body1)
	WriteFile(fs, "src/posts/01-test/meta.yaml", meta1)
	WriteFile(fs, "src/posts/02-test/body.md", body2)
	WriteFile(fs, "src/posts/02-test/meta.yaml", meta2)
	WriteFile(fs, "src/index.tmpl", index)
	var (
		err error
		out string
	)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out, err = ReadFile(fs, "build/index.html"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != html {
		t.Fatalf("Read:\n%v\nExpected:\n%v", out, html)
	}
}

// Ensures templates con include other templates.
func TestIncludeTemplates(t *testing.T) {
	gw, fs := Setup()
	body1 := "Post 1"
	meta1 := "date: 2012-09-07\nslug: post1"
	tmpl := `<html>
  <head>
    <title>{{.Title}}</title>
    {{template "head" .}}
  </head>
  <body>
    {{template "body" .}}
  </body>
</html>`
	index := `
{{define "head"}}<meta foo>{{end}}
{{define "body"}}{{range .Posts}}<div>{{.Body}}</div>{{end}}{{end}}`
	tmpl_post := `{{define "head"}}{{end}}{{define "body"}}{{end}}`
	html := `<html>
  <head>
    <title>Test blog</title>
    <meta foo>
  </head>
  <body>
    <div><p>Post 1</p>
</div>
  </body>
</html>`
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/global.tmpl", tmpl)
	WriteFile(fs, "src/templates/post.tmpl", tmpl_post)
	WriteFile(fs, "src/posts/01-test/body.md", body1)
	WriteFile(fs, "src/posts/01-test/meta.yaml", meta1)
	WriteFile(fs, "src/index.tmpl", index)
	var (
		err error
		out string
	)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out, err = ReadFile(fs, "build/index.html"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != html {
		t.Fatalf("Read:\n%v\nExpected:\n%v", out, html)
	}
}

// Ensures pages are rendered from a master template
func TestTemplateHierarchy(t *testing.T) {
	gw, fs := Setup()
	body1 := "Post 1"
	meta1 := "date: 2012-09-07\nslug: post1"
	body2 := "Post 2"
	meta2 := "date: 2012-09-08\nslug: post2"
	tmpl_base := "<html>{{template \"h\" .}}{{template \"b\" .}}</html>"
	tmpl_post := "{{define \"h\"}}{{end}}{{define \"b\"}}[{{.Post.Body}}]{{end}}"
	tmpl_indx := "{{define \"h\"}}[head]{{end}}{{define \"b\"}}{{range .Posts}}[{{.Body}}]{{end}}{{end}}"
	html_indx := "<html>[head][<p>Post 1</p>\n][<p>Post 2</p>\n]</html>"
	html_post := "<html>[<p>Post 1</p>\n]</html>"
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/global.tmpl", tmpl_base)
	WriteFile(fs, "src/templates/post.tmpl", tmpl_post)
	WriteFile(fs, "src/posts/01-test/body.md", body1)
	WriteFile(fs, "src/posts/01-test/meta.yaml", meta1)
	WriteFile(fs, "src/posts/02-test/body.md", body2)
	WriteFile(fs, "src/posts/02-test/meta.yaml", meta2)
	WriteFile(fs, "src/index.tmpl", tmpl_indx)
	var (
		err error
		out string
	)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out, err = ReadFile(fs, "build/index.html"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != html_indx {
		t.Errorf("Read:\n%v\nExpected:\n%v", out, html_indx)
	}
	if out, err = ReadFile(fs, "build/2012-09-07/post1/index.html"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != html_post {
		t.Errorf("Read:\n%v\nExpected:\n%v", out, html_post)
	}
}
