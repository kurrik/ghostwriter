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
	"os"
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

// Sensible defaults, for a sensible time.
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
	var (
		watch bool
		gw    *GhostWriter
		err   error
	)
	a := DefaultArgs()
	flag.StringVar(&a.src, "src", "src", "Path to src files.")
	flag.StringVar(&a.dst, "dst", "dst", "Build output directory.")
	flag.BoolVar(&watch, "watch", false, "Keep watching the source dir?")
	flag.Parse()
	gw = NewGhostWriter(&fauxfile.RealFilesystem{}, a)
	if watch {
		err = Watch(gw, a.src)
	} else {
		err = gw.Process()
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
