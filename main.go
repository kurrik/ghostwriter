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
	"flag"
	"fmt"
	"github.com/kurrik/fauxfile"
)

// Arguments, passed to the main executable.
type Args struct {
	src          string
	dst          string
	posts        string
	templates    string
	static       string
	config       string
	postTemplate string
	tagsTemplate string
}

func DefaultArgs() *Args {
	return &Args{
		src:          "src",
		dst:          "dst",
		posts:        "posts",
		templates:    "templates",
		static:       "static",
		config:       "config.yaml",
		postTemplate: "post.tmpl",
		tagsTemplate: "tags.tmpl",
	}
}

// Main routine.
func main() {
	a := DefaultArgs()
	flag.StringVar(&a.src, "src", "src", "Path to src files.")
	flag.StringVar(&a.dst, "dst", "dst", "Build output directory.")
	flag.Parse()
	w := NewGhostWriter(&fauxfile.RealFilesystem{}, a)
	if err := w.Process(); err != nil {
		fmt.Println(err)
	}
}
