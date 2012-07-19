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
	"github.com/kurrik/go-fauxfile"
	"log"
)

type Configuration struct {
	source string
	build  string
	fs     fauxfile.Filesystem
}

type GhostWriter struct {
	config *Configuration
}

func (w *GhostWriter) Parse() error {
	log.Printf("Parsing directory %v", w.config.source)
	return nil
}

func main() {
	c := &Configuration{}
	flag.StringVar(&c.source, "source", "src", "Path to source files.")
	flag.StringVar(&c.build, "build", "build", "Build output directory.")
	flag.Parse()
	w := &GhostWriter{config: c}
	if err := w.Parse(); err != nil {
		fmt.Println(err)
	}
}
