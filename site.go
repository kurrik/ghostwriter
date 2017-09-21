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
	"sort"
	"text/template"
	"time"
)

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

// Returns a list of TagCount objects, sorted by count.
func (s *Site) TagCounts() TagCounts {
	counts := make(TagCounts, len(s.Tags))
	i := 0
	for tag, posts := range s.Tags {
		counts[i] = &TagCount{Tag: tag, Count: len(posts)}
		i++
	}
	sort.Sort(counts)
	return counts
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

// Returns any additional user-specified metadata.
func (s *Site) Metadata() map[string]string {
	return s.meta.Metadata
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
