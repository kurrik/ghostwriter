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
	if err = gw.renderPosts("posts"); err != nil {
		return
	}
	return
}

func (gw *GhostWriter) copyStatic(name string) (err error) {
	var (
		queue []string
		names []string
		p     string
		n     string
		src   string
		dst   string
		x     int
		i     os.FileInfo
		fsrc  fauxfile.File
		fdst  fauxfile.File
		b     []byte
		c     int
	)
	queue = []string{name}
	for len(queue) > 0 {
		p = queue[0]
		queue = queue[1:]
		src = filepath.Join(gw.args.src, p)
		dst = filepath.Join(gw.args.dst, p)
		if i, err = gw.fs.Stat(src); err != nil {
			if name == p {
				gw.log.Printf("Static dir not found %v\n", src)
				// Fail silently
				return nil
			}
			return
		}
		if i.IsDir() {
			if fsrc, err = gw.fs.Open(src); err != nil {
				return
			}
			if names, err = fsrc.Readdirnames(-1); err != nil {
				return
			}
			fsrc.Close()
			for x, n = range names {
				names[x] = filepath.Join(p, n)
			}
			queue = append(queue, names...)
			gw.log.Printf("Creating %v\n", dst)
			if err = gw.fs.Mkdir(dst, i.Mode()); err != nil {
				gw.log.Printf("Problem creating %v\n", dst)
				return
			}
		} else {
			gw.log.Printf("Copying %v to %v\n", src, dst)
			if fdst, err = gw.fs.Create(dst); err != nil {
				return
			}
			fdst.Chmod(i.Mode())
			if fsrc, err = gw.fs.Open(src); err != nil {
				fdst.Close()
				return
			}
			b = make([]byte, 10*1024) // 10Kb
			for {
				c, err = fsrc.Read(b)
				if err == io.EOF {
					err = nil
					break
				}
				if err != nil {
					break
				}
				b = b[:c]
				c, err = fdst.Write(b)
				if err != nil {
					break
				}
			}
			fsrc.Close()
			fdst.Close()
			if err != nil {
				return
			}
		}
	}
	return
}

func (gw *GhostWriter) parseConfig(path string) (err error) {
	var (
		src string = filepath.Join(gw.args.src, path)
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

func (gw *GhostWriter) renderPosts(name string) (err error) {
	var (
		src   = filepath.Join(gw.args.src, name)
		fsrc  fauxfile.File
		names []string
		p     string
	)
	if fsrc, err = gw.fs.Open(src); err != nil {
		gw.log.Printf("Posts directory not found %v\n", src)
		// Fail silently
		return nil
	}
	names, err = fsrc.Readdirnames(-1)
	fsrc.Close()
	if err != nil {
		return
	}
	for _, p = range names {
		fmt.Printf("Processing %v\n", p)
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
