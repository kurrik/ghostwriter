// Copyright 2017 Arne Roomann-Kurrik
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
	"bytes"
	"fmt"
	"net/url"
	"path"
	"strings"
	"text/template"
	"time"
)

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

// Resolves paths for a list of inputs.
func (p *Post) resolvePaths(input []string) (output []string) {
	var (
		i        = 0
		postpath string
	)
	output = make([]string, len(input))
	postpath, _ = p.Path()
	for i = 0; i < len(input); i++ {
		if strings.HasPrefix(input[i], "/") {
			output[i] = input[i]
		} else {
			output[i] = path.Join(postpath, input[i])
		}
	}
	return
}

// Returns any script URLs corresponding with the post.
func (p *Post) Scripts() (s []string) {
	s = p.resolvePaths(p.meta.Scripts)
	return
}

// Returns any style URLs corresponding with the post.
func (p *Post) Styles() (s []string) {
	s = p.resolvePaths(p.meta.Styles)
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
