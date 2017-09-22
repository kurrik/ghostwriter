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
	"encoding/base64"
	"github.com/kurrik/fauxfile"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"testing"
)

// Set to true to display build output from tests.
const SHOW_OUTPUT = false

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
	if SHOW_OUTPUT {
		gw.log = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		gw.log = log.New(ioutil.Discard, "", log.LstdFlags)
	}
	return
}

func WriteFile(fs fauxfile.Filesystem, p string, data string) error {
	return WriteFileBytes(fs, p, []byte(data))
}

func WriteFileBytes(fs fauxfile.Filesystem, p string, data []byte) error {
	var (
		f   fauxfile.File
		err error
	)
	fs.MkdirAll(path.Dir(p), 0755)
	if f, err = fs.Create(p); err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		return err
	}
	return nil
}

func WriteBase64File(fs fauxfile.Filesystem, p string, base64data string) error {
	var (
		data []byte
		err  error
	)
	if data, err = base64.StdEncoding.DecodeString(base64data); err != nil {
		return err
	}
	return WriteFileBytes(fs, p, data)
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
tagsformat: /tags/{{.Tag}}
recentcount: 5`

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
----------------
This is just {{textcontent "<a href='foo'>text content</a>"}} sans HTML.`

const POST_2_META = `
date: 2012-09-09
slug: hello-again
title: Hello Again!
tags:
  - hello
scripts:
  - foo.js
  - /bar.js
styles:
  - foo.css`

const POST_2_MD = `
This is a <a href="{{link "01-test"}}">link</a> to a post.
<img src="{{link "01-test/img.png"}}" />

<!--BREAK-->

This is content after the break`

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
    <div>Updated on {{timeformat .SureDate "2006-01-02T15:04:05Z07:00"}}</div>
    <div>{{.Snippet}}</div>
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
    <div>Updated on 2012-09-09T00:00:00Z</div>
    <div>
      <p>
        This is a <a href="/2012-09-07/hello-world">link</a> to a post.
        <img src="/2012-09-07/hello-world/img.png" />
      </p>
    </div>
    <h2>Hello World!</h2>
    <div>Updated on 2012-09-07T00:00:00Z</div>
    <div>
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
    <div>Updated on 2012-09-07T00:00:00Z</div>
    <div>
    </div>
  </body>
</html>`

const POST_TMPL = `
{{define "head"}}
  <title>{{.Site.Title}} - {{.Post.Title}}</title>
  <link rel="canonical" href="{{.Post.Permalink}}" />
  {{range .Post.Styles}}
    <link rel="stylesheet" href="{{.}}" />
  {{end}}
{{end}}
{{define "body"}}
  <h1>{{.Post.Title}}</h1>
  <div>{{.Post.Body}}</div>
  {{if .Post.Next}}<a href="{{.Post.Next.Path}}">Next Post</a>{{end}}
  {{if .Post.Prev}}<a href="{{.Post.Prev.Path}}">Prev Post</a>{{end}}
  {{range .Post.Scripts}}
    <script src="{{.}}"></script>
  {{end}}
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
      <p>This is just text content sans HTML.</p>
    </div>
    <a href="/2012-09-09/hello-again">Next Post</a>
  </body>
</html>`

const POST_1_EMPTY_HTML = `
<!DOCTYPE html>
<html>
  <head>
    <title>Test blog - Hello World!</title>
    <link rel="canonical" href="http://www.example.com/2012-09-07/hello-world" />
  </head>
  <body>
    <h1>Hello World!</h1>
    <div>
    </div>
  </body>
</html>`

const POST_2_HTML = `
<!DOCTYPE html>
<html>
  <head>
    <title>Test blog - Hello Again!</title>
    <link rel="canonical" href="http://www.example.com/2012-09-09/hello-again" />
    <link rel="stylesheet" href="/2012-09-09/hello-again/foo.css" />
  </head>
  <body>
    <h1>Hello Again!</h1>
    <div>
      <p>
        This is a <a href="/2012-09-07/hello-world">link</a> to a post.
        <img src="/2012-09-07/hello-world/img.png" />
      </p>
      <!--BREAK-->
      <p>This is content after the break</p>
    </div>
    <a href="/2012-09-07/hello-world">Prev Post</a>
    <script src="/2012-09-09/hello-again/foo.js"></script>
    <script src="/bar.js"></script>
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
      <!--BREAK-->
      <p>This is content after the break</p>
    </div>
    <h2>Hello World!</h2>
    <div>
      <p>This is a fake post, for testing.</p>
      <h2>This is markdown</h2>
      <p>This is just text content sans HTML.</p>
    </div>
  </body>
</html>`

const BASE64_IMAGE = `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3AMGBSkd6mhiIQAAOlZJREFUeNrtnWesFWUax++uigVsYEUF7A1RERsWrIioWIKIIiLC2hAVVISACCiIBaMrFqwRFUUIKCgYewlYid3Yo2viB/eD2Y1fTDa5+/M++jjOae+8887MOec+/w+blXvOnJl5//POU/9PS6shNH799dfvv//+s88+e/311xctWnTXXXfNmDFj3Lhx55133mmnnda/f/+DDz64Z8+eO+ywQ9euXTt37typU6d111137bXXbvkDf//73zt06LDBBhtsuummW2+99Y477tirV69DDjnk+OOPP+ussy699NLrrrvu3nvvXb58+Zo1a7777ju7545osVvgh2+//RaqvfLKKxD69ttvnzRp0qhRo0444YQ+ffr06NGjY8eOLUnwt3Jw/O7666+/yy67HHnkkSNGjJg2bdojjzzy6quv8rDZGhndE0M26aeeemrevHnTp0+/6KKL2KHZaHfeeeeknM4TvBN4AC644IJbb7112bJln3zyiS2l0f0vYLdesmQJW/UVV1xxzjnnDBw48KCDDoLWWBQpyYdZsuWWW2KT7LXXXhwTInJwHhssE7bk0aNHX3jhhRdffDFWymVtGDNmDM/V+eeff+6555555pmnnnoqZky/fv14dey2225QmR090Qnw0xzh6quvXrBgwZdffml0b4/4/PPPH3vssfHjxw8aNGj//ffv3r17UhopsL+xxQ888MCTTjoJG/2qq66aPXv2fffdh6nz3HPPvfHGGzxL/Bw2PZa99wn//PPPX3/99fvvv88L59lnn4W7t91224QJE3gkDjvssO23376mCYRXgOXDwzZlypSXX37Z6N78Nvf9998/cuRI+N2tW7dE/IZM7PR4mbibHOHaa6/FwnnmmWfeeecdbB6oXOylcQIfffTRihUrMGB4MxxwwAEbbrhhlcvZZJNN9t57b14mr732mtG9qbB69erJkydjDMDXtdZaqwoJCJJsvPHGGAxQAQMASwNas0+/9NJLX331VWNd9S+//IIzfdNNNw0ePJiXD5dWydbCypo4ceKHH35odG9U/PDDD7iY8JVdvAq/t9pqqz322OOYY4655JJL5s6d+8ILL/z4449NeUPee+89qI/lhkMCxcvejb59+95zzz1YTUb3hgF2Lc7fdtttV4ni22677aGHHopB8s9//vPtt99uh37L4sWL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`

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
	WriteFile(fs, "src/posts/01-test/subdir/content.png", content)
	WriteFile(fs, "src/posts/01-test/sub/dir/content.png", content)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out, err = ReadFile(fs, "build/2012-09-07/hello-world/content.png"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != content {
		t.Fatalf("Read:\n%v\nExpected:\n%v", out, content)
	}
	if out, err = ReadFile(fs, "build/2012-09-07/hello-world/subdir/content.png"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != content {
		t.Fatalf("Read:\n%v\nExpected:\n%v", out, content)
	}
	if out, err = ReadFile(fs, "build/2012-09-07/hello-world/sub/dir/content.png"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if out != content {
		t.Fatalf("Read:\n%v\nExpected:\n%v", out, content)
	}
}

// Ensures that empty post dirs don't raise errors.
func TestPostWithEmptyDirDoesNotRaiseError(t *testing.T) {
	var (
		err error
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/post.tmpl", "")
	WriteFile(fs, "src/posts/01-test/body.md", "")
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_1_META)
	fs.MkdirAll("src/posts/02-test", 0755)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
}

// Ensures that empty meta files don't raise errors.
func TestPostWithEmptyMetaIsNotParsed(t *testing.T) {
	var (
		err error
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/post.tmpl", "")
	WriteFile(fs, "src/posts/01-test/body.md", "")
	WriteFile(fs, "src/posts/01-test/meta.yaml", "")
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if _, ok := gw.site.Posts["01-test"]; ok {
		t.Fatalf("Empty meta should not parse into a post")
	}
}

// Ensures that missing bodies don't raise errors.
func TestPostWithMissingBodyIsValid(t *testing.T) {
	var (
		err error
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/root.tmpl", SITE_TMPL)
	WriteFile(fs, "src/templates/post.tmpl", POST_TMPL)
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_1_META)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	LooseCompareFile(t, fs, "build/2012-09-07/hello-world/index.html", POST_1_EMPTY_HTML)
}

// Ensures that files in the posts directory are not treated as post dirs.
func TestFileInPostsDirNotTreatedAsPostDirectory(t *testing.T) {
	var (
		err error
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/post.tmpl", "")
	WriteFile(fs, "src/posts/01-test/body.md", "")
	WriteFile(fs, "src/posts/01-test/meta.yaml", "")
	WriteFile(fs, "src/posts/junk.yaml", "")
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
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

const POST_INCLUDE_META = `
date: 2015-09-01
slug: include
title: Include`

const POST_VALID_INCLUDE = `
This post makes a valid include.
{{include "include.html"}}`

const POST_INVALID_INCLUDE = `
This post makes an invalid include.
{{include "foo.html"}}`

const INCL_CONTENT = `
This is included content including {{ mustache }}`

const POST_VALID_INCLUDE_HTML = `<!DOCTYPE html>
<html>
  <head>
  <title>Test blog - Include</title>
  <link rel="canonical" href="http://www.example.com/2015-09-01/include" />
  </head>
  <body>
    <h1>Include</h1>
    <div>
      <p>This post makes a valid include.</p>
      <p>This is included content including {{ mustache }}</p>
    </div>
  </body>
</html>`

const POST_INVALID_INCLUDE_HTML = `<!DOCTYPE html>
<html>
  <head>
  <title>Test blog - Include</title>
  <link rel="canonical" href="http://www.example.com/2015-09-01/include" />
  </head>
  <body>
    <h1>Include</h1>
    <div>
      <p>
        This post makes an invalid include.
        [[ERROR: Could not read src/posts/01-test/foo.html]]
      </p>
    </div>
  </body>
</html>`

func TestIncludeValid(t *testing.T) {
	var (
		err error
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/root.tmpl", SITE_TMPL)
	WriteFile(fs, "src/templates/post.tmpl", POST_TMPL)
	WriteFile(fs, "src/posts/01-test/body.md", POST_VALID_INCLUDE)
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_INCLUDE_META)
	WriteFile(fs, "src/posts/01-test/include.html", INCL_CONTENT)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	LooseCompareFile(t, fs, "build/2015-09-01/include/index.html", POST_VALID_INCLUDE_HTML)
}

func TestIncludeInvalid(t *testing.T) {
	var (
		err error
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/root.tmpl", SITE_TMPL)
	WriteFile(fs, "src/templates/post.tmpl", POST_TMPL)
	WriteFile(fs, "src/posts/01-test/body.md", POST_INVALID_INCLUDE)
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_INCLUDE_META)
	WriteFile(fs, "src/posts/01-test/include.html", INCL_CONTENT)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	LooseCompareFile(t, fs, "build/2015-09-01/include/index.html", POST_INVALID_INCLUDE_HTML)
}

const IMAGEMETA_SITE_TMPL = `
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
{{define "gallery"}}{{end}}
{{define "body"}}{{end}}`

const POST_IMAGEMETA_META = `
date: 2017-03-16
slug: imagemeta
title: Imagemeta`

const POST_VALID_IMAGEMETA = `
This post makes a valid imagemeta load.
{{ $image01 := imagemeta "image01.png" }}
{{ $thumb01 := imagemeta "image01_thumb.png" }}
{{ $entry01 := map "Image" $image01 "Thumb" $thumb01 "Description" "An image!" }}
{{ $image02 := imagemeta "image02.png" }}
{{ $thumb02 := imagemeta "image02_thumb.png" }}
{{ $entry02 := map "Image" $image02 "Thumb" $thumb02 "Description" "Another image!" }}
{{template "gallery" (slice $entry01 $entry02)}}`

const IMAGEMETA_TEMPLATE_GALLERY = `{{define "gallery"}}
{{range .}}
  <a href="{{.Image.Path}}">
    <img src="{{.Thumb.Path}}" width="{{.Thumb.Width}}" height="{{.Thumb.Height}}" />
  </a>
  {{.Description}} - {{.Image.Width}}x{{.Image.Height}}
{{end}}
{{end}}`

const POST_VALID_IMAGEMETA_HTML = `<!DOCTYPE html>
<html>
  <head>
  <title>Test blog - Imagemeta</title>
  <link rel="canonical" href="http://www.example.com/2017-03-16/imagemeta" />
  </head>
  <body>
    <h1>Imagemeta</h1>
    <div>
      <p>This post makes a valid imagemeta load.</p>
      <p>
        <a href="/2017-03-16/imagemeta/image01.png">
          <img src="/2017-03-16/imagemeta/image01_thumb.png" width="250" height="340" />
        </a>
        An image! - 250x340
      </p>
      <p>
        <a href="/2017-03-16/imagemeta/image02.png">
          <img src="/2017-03-16/imagemeta/image02_thumb.png" width="250" height="340" />
        </a>
        Another image! - 250x340
      </p>
    </div>
  </body>
</html>`

func TestImagemetaValid(t *testing.T) {
	var (
		err error
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/root.tmpl", IMAGEMETA_SITE_TMPL)
	WriteFile(fs, "src/templates/post.tmpl", POST_TMPL)
	WriteFile(fs, "src/templates/gallery.tmpl", IMAGEMETA_TEMPLATE_GALLERY)
	WriteFile(fs, "src/posts/01-test/body.md", POST_VALID_IMAGEMETA)
	WriteFile(fs, "src/posts/01-test/meta.yaml", POST_IMAGEMETA_META)
	WriteBase64File(fs, "src/posts/01-test/image01.png", BASE64_IMAGE)
	WriteBase64File(fs, "src/posts/01-test/image01_thumb.png", BASE64_IMAGE)
	WriteBase64File(fs, "src/posts/01-test/image02.png", BASE64_IMAGE)
	WriteBase64File(fs, "src/posts/01-test/image02_thumb.png", BASE64_IMAGE)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	LooseCompareFile(t, fs, "build/2017-03-16/imagemeta/index.html", POST_VALID_IMAGEMETA_HTML)
}

const YAMLTEMPLATE_META = `
date: 2017-03-19
slug: yamltemplate
title: Yamltemplate`

const YAMLTEMPLATE_BODY = `
{{define "testdata"}}
gallery:
  - image: {{imagemeta "image01.png" | tojson}}
    thumb: {{imagemeta "image01_thumb.png" | tojson}}
    description: Image One
  - image: {{imagemeta "image02.png" | tojson}}
    thumb: {{imagemeta "image02_thumb.png" | tojson}}
    description: Image Two
{{end}}

{{define "rendertestdata"}}
{{range .gallery}}
  Description: {{.description}}
  Image: {{.image.Path}}
  Thumb: {{.thumb.Path}}
{{end}}
{{end}}

This post makes a valid yamltemplate reference.
{{template "rendertestdata" (yamltemplate "testdata")}}`

const YAMLTEMPLATE_VALID_HTML = `<!DOCTYPE html>
<html>
  <head>
  <title>Test blog - Yamltemplate</title>
  <link rel="canonical" href="http://www.example.com/2017-03-19/yamltemplate" />
  </head>
  <body>
    <h1>Yamltemplate</h1>
    <div>
      <p>This post makes a valid yamltemplate reference.</p>
      <p>
        Description: Image One
        Image: /2017-03-19/yamltemplate/image01.png
        Thumb: /2017-03-19/yamltemplate/image01_thumb.png
      </p>
      <p>
        Description: Image Two
        Image: /2017-03-19/yamltemplate/image02.png
        Thumb: /2017-03-19/yamltemplate/image02_thumb.png
      </p>
    </div>
  </body>
</html>`

func TestYamltemplateValid(t *testing.T) {
	var (
		err error
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/root.tmpl", SITE_TMPL)
	WriteFile(fs, "src/templates/post.tmpl", POST_TMPL)
	WriteFile(fs, "src/posts/01-test/body.md", YAMLTEMPLATE_BODY)
	WriteFile(fs, "src/posts/01-test/meta.yaml", YAMLTEMPLATE_META)
	WriteBase64File(fs, "src/posts/01-test/image01.png", BASE64_IMAGE)
	WriteBase64File(fs, "src/posts/01-test/image01_thumb.png", BASE64_IMAGE)
	WriteBase64File(fs, "src/posts/01-test/image02.png", BASE64_IMAGE)
	WriteBase64File(fs, "src/posts/01-test/image02_thumb.png", BASE64_IMAGE)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	LooseCompareFile(t, fs, "build/2017-03-19/yamltemplate/index.html", YAMLTEMPLATE_VALID_HTML)
}

const POSTIMAGES_META = `
date: 2017-09-17
slug: postimages
title: Post Images
images:
  image01:
    src: "image01.png"
    metadata:
      href: "http://example.com/foo"
      alt: "Test alt"
  image02:
    src: "image02.png"
    variants:
      thumb:
        src: "image02_thumb.png"
    metadata:
      alt: "Alt2"
`

const RENDERIMAGE_TMPL = `
{{define "renderimage"}}
  {{- with .Metadata.href}}<a href="{{.}}">{{end -}}
  {{- $img := or .Variants.thumb .Data -}}
  <img src="{{$img.Path}}" width="{{$img.Width}}" height="{{$img.Height}}" {{with .Metadata.alt}}alt="{{.}}"{{end}}/>
  {{- with .Metadata.href}}</a>{{end -}}
{{end}}
`

const POSTIMAGES_BODY = `
This post has valid image data associated with its metadata.

{{template "renderimage" (.Image "image01")}}

{{template "renderimage" (.Image "image02")}}`

const POSTIMAGES_VALID_HTML = `<!DOCTYPE html>
<html>
  <head>
  <title>Test blog - Post Images</title>
  <link rel="canonical" href="http://www.example.com/2017-09-17/postimages" />
  </head>
  <body>
    <h1>Post Images</h1>
    <div>
      <p>This post has valid image data associated with its metadata.</p>
      <p><a href="http://example.com/foo"><img src="/2017-09-17/postimages/image01.png" width="250" height="340" alt="Test alt"/></a></p>
      <p><img src="/2017-09-17/postimages/image02_thumb.png" width="250" height="340" alt="Alt2"/></p>
    </div>
  </body>
</html>`

func TestPostImagesValid(t *testing.T) {
	var (
		err error
	)
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", SITE_META)
	WriteFile(fs, "src/templates/root.tmpl", SITE_TMPL)
	WriteFile(fs, "src/templates/post.tmpl", POST_TMPL)
	WriteFile(fs, "src/templates/renderimage.tmpl", RENDERIMAGE_TMPL)
	WriteFile(fs, "src/posts/01-test/body.md", POSTIMAGES_BODY)
	WriteFile(fs, "src/posts/01-test/meta.yaml", POSTIMAGES_META)
	WriteBase64File(fs, "src/posts/01-test/image01.png", BASE64_IMAGE)
	WriteBase64File(fs, "src/posts/01-test/image01_thumb.png", BASE64_IMAGE)
	WriteBase64File(fs, "src/posts/01-test/image02.png", BASE64_IMAGE)
	WriteBase64File(fs, "src/posts/01-test/image02_thumb.png", BASE64_IMAGE)
	if err = gw.Process(); err != nil {
		t.Fatalf("Error: %v", err)
	}
	LooseCompareFile(t, fs, "build/2017-09-17/postimages/index.html", POSTIMAGES_VALID_HTML)
}
