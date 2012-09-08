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
	"io"
	"launchpad.net/goyaml"
	"log"
	"os"
	"path/filepath"
)

type Args struct {
	src string
	dst string
}

type Configuration struct {
	fs fauxfile.Filesystem
}

type GhostWriter struct {
	args   *Args
	fs     fauxfile.Filesystem
	log    *log.Logger
	config map[interface{}]interface{}
}

func NewGhostWriter(fs fauxfile.Filesystem, args *Args) *GhostWriter {
	gw := &GhostWriter{
		args: args,
		fs:   fs,
		log:  log.New(os.Stderr, "", log.LstdFlags),
	}
	return gw
}

func (gw *GhostWriter) Process() (err error) {
	if err = gw.parseConfig("config.yaml"); err != nil {
		return
	}
	if err = gw.copyStatic("static"); err != nil {
		return
	}
	return
}

func (gw *GhostWriter) copyStatic(path string) (err error) {
	var (
		queue []string
		names []string
		p     string
		n     string
		s     string
		d     string
		x     int
		i     os.FileInfo
		f     fauxfile.File
		f2    fauxfile.File
		b     []byte
		c     int
	)
	queue = []string{path}
	for len(queue) > 0 {
		p = queue[0]
		queue = queue[1:]
		s = filepath.Join(gw.args.src, p)
		d = filepath.Join(gw.args.dst, p)
		if i, err = gw.fs.Stat(s); err != nil {
			return
		}
		if i.IsDir() {
			if f, err = gw.fs.Open(s); err != nil {
				return
			}
			if names, err = f.Readdirnames(-1); err != nil {
				return
			}
			f.Close()
			for x, n = range names {
				names[x] = filepath.Join(p, n)
			}
			queue = append(queue, names...)
			gw.log.Printf("Creating %v\n", d)
			if err = gw.fs.Mkdir(d, i.Mode()); err != nil {
				gw.log.Printf("Problem creating %v\n", d)
				return
			}
		} else {
			gw.log.Printf("Copying %v to %v\n", s, d)
			if f, err = gw.fs.Create(d); err != nil {
				return
			}
			f.Chmod(i.Mode())
			if f2, err = gw.fs.Open(s); err != nil {
				return
			}
			b = make([]byte, 10*1024) // 10Kb
			for {
				c, err = f2.Read(b)
				if err == io.EOF {
					break
				}
				if err != nil {
					f.Close()
					f2.Close()
					return
				}
				b = b[:c]
				c, err = f.Write(b)
				if err != nil {
					f.Close()
					f2.Close()
					return
				}
			}
			f.Close()
			f2.Close()
		}
	}
	return
}

func (gw *GhostWriter) parseConfig(path string) (err error) {
	var (
		src  = filepath.Join(gw.args.src, path)
		f    fauxfile.File
		info os.FileInfo
		data []byte
	)
	gw.log.Printf("Parsing config %v\n", src)
	if f, err = gw.fs.Open(src); err != nil {
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
	flag.StringVar(&a.src, "src", "src", "Path to src files.")
	flag.StringVar(&a.dst, "dst", "dst", "Build output directory.")
	flag.Parse()
	w := NewGhostWriter(&fauxfile.RealFilesystem{}, a)
	if err := w.Process(); err != nil {
		fmt.Println(err)
	}
}
