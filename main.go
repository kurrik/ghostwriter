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
	"launchpad.net/goyaml"
	"log"
	"os"
	"path/filepath"
)

type Args struct {
	source string
	build  string
}

type Configuration struct {
	fs fauxfile.Filesystem
}

type GhostWriter struct {
	args   *Args
	fs     fauxfile.Filesystem
	config map[interface{}]interface{}
}

func NewGhostWriter(fs fauxfile.Filesystem, args *Args) *GhostWriter {
	gw := &GhostWriter{args: args, fs: fs}
	return gw
}

func (gw *GhostWriter) Process() (err error) {
	log.Printf("Parsing directory %v", gw.args.source)
	gw.parseConfig(filepath.Join(gw.args.source, "config.yaml"))
	return nil
}

func (gw *GhostWriter) parseConfig(path string) (err error) {
	var (
		f    fauxfile.File
		info os.FileInfo
		data []byte
	)
	if f, err = gw.fs.Open(path); err != nil {
		return
	}
	if info, err = f.Stat(); err != nil {
		return
	}
	gw.config = make(map[interface{}]interface{})
	data = make([]byte, info.Size())
	if _, err = f.Read(data); err != nil {
		return
	}
	if err = goyaml.Unmarshal(data, gw.config); err != nil {
		return
	}
	return
}

func main() {
	a := &Args{}
	flag.StringVar(&a.source, "source", "src", "Path to source files.")
	flag.StringVar(&a.build, "build", "build", "Build output directory.")
	flag.Parse()
	w := NewGhostWriter(&fauxfile.RealFilesystem{}, a)
	if err := w.Process(); err != nil {
		fmt.Println(err)
	}
}
