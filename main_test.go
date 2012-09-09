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

func TestProcess(t *testing.T) {
	gw, fs := Setup()
	WriteFile(fs, "/home/test/src/config.yaml", "")
	if err := gw.Process(); err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
}

// Ensures that config files are parsed and values pulled out.
func TestParseConfig(t *testing.T) {
	gw, fs := Setup()
	conf := `
title: Test blog
root: www.example.com
pathformat: /{{date}}/{{slug}}
dateformat: "%Y-%m-%d"`
	WriteFile(fs, "/home/test/src/config.yaml", conf)
	if err := gw.parseConfig("config.yaml"); err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}
	if gw.site.Title != "Test blog" {
		t.Errorf("Bad title, got %v", gw.site.Title)
	}
	if gw.site.Root != "www.example.com" {
		t.Errorf("Bad root, got %v", gw.site.Root)
	}
	if gw.site.PathFormat != "/{{date}}/{{slug}}" {
		t.Errorf("Bad path format, got %v", gw.site.PathFormat)
	}
	if gw.site.DateFormat != "%Y-%m-%d" {
		t.Errorf("Bad date format, got %v", gw.site.DateFormat)
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

func TestRenderContent(t *testing.T) {
	gw, fs := Setup()
	conf := `
title: Test blog
root: www.example.com
pathformat: /{{date}}/{{slug}}
dateformat: \%Y-%m-%d`
	body := `
Hello World
===========
This is a fake post, for testing.

This is markdown
----------------`
	tmpl := `<!DOCTYPE html>
<html>
  <head>
    <title>{{site.title}} - {{post.slug}}</title>
    <link rel="canonical" href="{{post.permalink}}" />
  </head>
  <body>
    {{post.body}}
  </body>
</html>`
	meta := `
date: 2012-09-07
slug: hello-world
title: Hello World!
tags:
  - hello
  - world`
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
	WriteFile(fs, "src/config.yaml", conf)
	WriteFile(fs, "src/templates/post.tmpl", tmpl)
	WriteFile(fs, "src/posts/01-test/body.md", body)
	WriteFile(fs, "src/posts/01-test/meta.yaml", meta)
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
