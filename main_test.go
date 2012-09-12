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
	fs.Mkdir("build", 0755)
	gw = NewGhostWriter(fs, &Args{
		src: "src",
		dst: "build",
	})
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
	if err := gw.parseSiteMeta("config.yaml"); err != nil {
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
	gw.Process()
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
{{.Post.Body}}
  </body>
</html>`
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
	WriteFile(fs, "src/templates/post.tmpl", tmpl)
	WriteFile(fs, "src/posts/01-test/body.md", body)
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_META)
	var (
		err error
		out string
	)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out, err = ReadFile(fs, "build/2012-09-07/hello-world"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != html {
		t.Fatalf("Read: %v, Expected: %v", out, html)
	}
}
