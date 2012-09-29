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
	"strings"
	"testing"
)

func LooseCompare(t *testing.T, a string, b string) bool {
	a = strings.Replace(a, " ", "", -1)
	a = strings.Replace(a, "\n", "", -1)
	b = strings.Replace(b, " ", "", -1)
	b = strings.Replace(b, "\n", "", -1)
	if a != b {
		t.Logf("LooseCompare diff:\n%v\n%v", a, b)
		return false
	}
	return true
}

func LooseCompareFile(t *testing.T, fs *fauxfile.MockFilesystem, path string, gold string) {
	var (
		err error
		out string
	)
	if out, err = ReadFile(fs, path); err != nil {
		t.Errorf("Error reading: %v", err)
		return
	}
	if !LooseCompare(t, out, gold) {
		t.Errorf("Read (%v):\n%v\nExpected:\n%v", path, out, gold)
	}
}

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
dateformat: "2006-01-02"
tagsformat: /tags/{{.Tag}}`

const POST_1_META = `
date: 2012-09-07
slug: hello-world
title: Hello World!
tags:
  - hello
  - world`

const POST_1_MD = `
This is a fake post, for testing.

This is markdown
----------------`

const POST_2_META = `
date: 2012-09-09
slug: hello-again
title: Hello Again!
tags:
  - hello`

const POST_2_MD = `
This is a <a href="{{link "01-test"}}">link</a> to a post.
<img src="{{link "01-test/img.png"}}" />`

const SITE_TMPL = `
<!DOCTYPE html>
<html>
  <head>
    {{template "head" .}}
  </head>
  <body>
    {{template "body" .}}
  </body>
</html>
{{define "head"}}
  <title>{{.Site.Title}}</title>
{{end}}
{{define "body"}}{{end}}`

const TAGS_TMPL = `
{{define "body"}}
  <h1>Posts tagged with {{.Tag}}</h1>
  {{range .Posts}}
    <h2>{{.Title}}</h2>
    <div>{{.Body}}</div>
  {{end}}
{{end}}`

const TAG_HELLO_HTML = `
<!DOCTYPE html>
<html>
  <head>
    <title>Test blog</title>
  </head>
  <body>
    <h1>Posts tagged with hello</h1>
    <h2>Hello Again!</h2>
    <div>
      <p>
        This is a <a href="/2012-09-07/hello-world">link</a> to a post.
        <img src="/2012-09-07/hello-world/img.png" />
      </p>
    </div>
    <h2>Hello World!</h2>
    <div>
      <p>This is a fake post, for testing.</p>
      <h2>This is markdown</h2>
    </div>
  </body>
</html>`

const TAG_WORLD_HTML = `
<!DOCTYPE html>
<html>
  <head>
    <title>Test blog</title>
  </head>
  <body>
    <h1>Posts tagged with world</h1>
    <h2>Hello World!</h2>
    <div>
      <p>This is a fake post, for testing.</p>
      <h2>This is markdown</h2>
    </div>
  </body>
</html>`

const POST_TMPL = `
{{define "head"}}
  <title>{{.Site.Title}} - {{.Post.Title}}</title>
  <link rel="canonical" href="{{.Post.Permalink}}" />
{{end}}
{{define "body"}}
  <h1>{{.Post.Title}}</h1>
  <div>{{.Post.Body}}</div>
{{end}}`

const POST_1_HTML = `
<!DOCTYPE html>
<html>
  <head>
    <title>Test blog - Hello World!</title>
    <link rel="canonical" href="http://www.example.com/2012-09-07/hello-world" />
  </head>
  <body>
    <h1>Hello World!</h1>
    <div>
      <p>This is a fake post, for testing.</p>
      <h2>This is markdown</h2>
    </div>
  </body>
</html>`

const POST_2_HTML = `
<!DOCTYPE html>
<html>
  <head>
    <title>Test blog - Hello Again!</title>
    <link rel="canonical" href="http://www.example.com/2012-09-09/hello-again" />
  </head>
  <body>
    <h1>Hello Again!</h1>
    <div>
      <p>
        This is a <a href="/2012-09-07/hello-world">link</a> to a post.
        <img src="/2012-09-07/hello-world/img.png" />
      </p>
    </div>
  </body>
</html>`

const INDEX_TMPL = `
{{define "body"}}
  <h1>{{.Site.Title}}</h1>
  {{range .Site.RecentPosts}}
    <h2>{{.Title}}</h2>
    <div>{{.Body}}</div>
  {{end}}
{{end}}`

const INDEX_HTML = `
<!DOCTYPE html>
<html>
  <head>
    <title>Test blog</title>
  </head>
  <body>
    <h1>Test blog</h1>
    <h2>Hello Again!</h2>
    <div>
      <p>
       This is a <a href="/2012-09-07/hello-world">link</a> to a post.
       <img src="/2012-09-07/hello-world/img.png" />
      </p>
    </div>
    <h2>Hello World!</h2>
    <div>
      <p>This is a fake post, for testing.</p>
      <h2>This is markdown</h2>
    </div>
  </body>
</html>`

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

// Ensures that post meta files are parsed and values pulled out.
func TestParsePostMeta(t *testing.T) {
	gw, fs := Setup()
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_1_META)
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

// Ensures post content (images, etc) are copied to build dir.
func TestPostContentCopied(t *testing.T) {
	var (
		err     error
		out     string
		content = "Content!"
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/post.tmpl", "")
	WriteFile(fs, "src/posts/01-test/body.md", "")
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_1_META)
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

// Ensures a complex site is rendered
func TestProcess(t *testing.T) {
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/root.tmpl", SITE_TMPL)
	WriteFile(fs, "src/templates/post.tmpl", POST_TMPL)
	WriteFile(fs, "src/templates/tags.tmpl", TAGS_TMPL)
	WriteFile(fs, "src/posts/01-test/body.md", POST_1_MD)
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_1_META)
	WriteFile(fs, "src/posts/01-test/img.png", "")
	WriteFile(fs, "src/posts/02-test/body.md", POST_2_MD)
	WriteFile(fs, "src/posts/02-test/meta.yaml", POST_2_META)
	WriteFile(fs, "src/index.tmpl", INDEX_TMPL)
	if err := gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	LooseCompareFile(t, fs, "build/index.html", INDEX_HTML)
	LooseCompareFile(t, fs, "build/2012-09-07/hello-world/index.html", POST_1_HTML)
	LooseCompareFile(t, fs, "build/2012-09-09/hello-again/index.html", POST_2_HTML)
	LooseCompareFile(t, fs, "build/tags/hello/index.html", TAG_HELLO_HTML)
	LooseCompareFile(t, fs, "build/tags/world/index.html", TAG_WORLD_HTML)
}