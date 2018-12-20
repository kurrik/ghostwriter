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
	"path/filepath"
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
	images  map[string]*Image
}

func NewPost(id string, srcDir string, site *Site) *Post {
	return &Post{
		Id:     id,
		SrcDir: srcDir,
		site:   site,
	}
}

// Parses post metadata at the supplied path and initializes the Post structure.
// If any fields are invalid, err will be non-nil.
func (p *Post) ParseMeta(gw *GhostWriter, path string) (err error) {
	src := filepath.Join(gw.args.src, path)
	p.meta = &PostMeta{}
	if err = gw.unyaml(src, p.meta); err != nil {
		return
	}
	if p.meta.Date == "" {
		err = fmt.Errorf("Post meta must include date")
		return
	}
	if p.meta.Slug == "" {
		err = fmt.Errorf("Post meta must include slug")
		return
	}
	if p.meta.Title == "" {
		err = fmt.Errorf("Post meta must include title")
		return
	}
	p.loadImageData(gw)
	return
}

// Attempts to load ImageData data for images associated with the post metadata.
func (p *Post) loadImageData(gw *GhostWriter) (err error) {
	var dstPath string
	if dstPath, err = p.Path(); err != nil {
		return
	}
	p.images = map[string]*Image{}
	for k, imageMeta := range p.meta.Images {
		if p.images[k], err = NewImage(gw, imageMeta, p.SrcDir, dstPath, p.site.Root()); err != nil {
			err = fmt.Errorf("Could not load image metadata: %v", err)
			return
		}
	}
	return
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

// Returns whether user-specified metadata exists
func (p *Post) HasMetadata(key string) (exists bool) {
	_, exists = p.meta.Metadata[key]
	return
}

// Returns any additional user-specified metadata.
func (p *Post) Metadata() map[string]string {
	return p.meta.Metadata
}

// Resolve a single path.
func (p *Post) resolvePath(input string) (output string) {
	var postpath string
	if strings.HasPrefix(input, "/") {
		output = input
	} else {
		postpath, _ = p.Path()
		output = path.Join(postpath, input)
	}
	return
}

// Resolves paths for a list of inputs.
func (p *Post) resolvePathsArray(input []string) (output []string) {
	var i = 0
	output = make([]string, len(input))
	for i = 0; i < len(input); i++ {
		output[i] = p.resolvePath(input[i])
	}
	return
}

// Returns any script URLs corresponding with the post.
func (p *Post) Scripts() (s []string) {
	s = p.resolvePathsArray(p.meta.Scripts)
	return
}

// Returns any style URLs corresponding with the post.
func (p *Post) Styles() (s []string) {
	s = p.resolvePathsArray(p.meta.Styles)
	return
}

// Returns image metadata for all images associated with the post.
func (p *Post) Images() map[string]*Image {
	return p.images
}

// Returns a single image associated with the post, by key.
func (p *Post) Image(key string) (out *Image, err error) {
	var exists bool
	if out, exists = p.images[key]; !exists {
		err = fmt.Errorf("Could not get image with key %v from post", key)
		return
	}
	return
}

// Returns a single image associated with the post, by key. Errors return nil.
func (p *Post) ImageIfExists(key string) (out *Image) {
	var exists bool
	if out, exists = p.images[key]; !exists {
		return nil
	}
	return
}

// Returns a list of images corresponding to the supplied keys.  If a key is
// invalid, it is omitted from the output array.
func (p *Post) ImageList(keys ...string) (out []*Image) {
	out = []*Image{}
	for _, key := range keys {
		img, err := p.Image(key)
		if img != nil && err == nil {
			out = append(out, img)
		}
	}
	return
}

// Returns a single image associated with the post, by key.
func (p *Post) HasImage(key string) (exists bool) {
	_, exists = p.images[key]
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
