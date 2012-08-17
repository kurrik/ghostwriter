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
	"io"
	"fmt"
	"github.com/kurrik/fauxfile"
	"os"
	"path/filepath"
	"testing"
)

func PrintFs(fs fauxfile.Filesystem) {
	var (
		dirs  []string
		path  string
		f     fauxfile.File
		fi    os.FileInfo
		files []os.FileInfo
	)
	dirs = append(dirs, "/")
	for len(dirs) > 0 {
		path = dirs[0]
		dirs = dirs[1:]
		f, _ = fs.Open(path)
		fi, _ = f.Stat()
		files, _ = f.Readdir(100)
		for _, fi = range files {
			name := filepath.Join(path, fi.Name())
			fmt.Printf("%-30v %v %v\n", name, fi.Mode(), fi.IsDir())
			if fi.IsDir() {
				dirs = append(dirs, name)
			}
		}
	}
}

func Setup() (gw *GhostWriter, fs fauxfile.Filesystem) {
	fs = fauxfile.NewMockFilesystem()
	fs.MkdirAll("/home/test", 0755)
	fs.Chdir("/home/test")
	fs.Mkdir("src", 0755)
	fs.Mkdir("build", 0755)
	gw = NewGhostWriter(fs, &Args{
		source: "src",
		build:  "build",
	})
	return
}

func WriteFile(fs fauxfile.Filesystem, path string, data string) error {
	var (
		f   fauxfile.File
		err error
	)
	if f, err = fs.Create(path); err != nil {
		return err
	}
	if _, err = f.Write([]byte(data)); err != nil {
		return err
	}
	f.Close()
	return nil
}

func ReadFile(fs fauxfile.Filesystem, path string) (data string, err error) {
	var (
		f  fauxfile.File
		fi os.FileInfo
	)
	if f, err = os.Open(path); err != nil {
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

func TestProcess(t *testing.T) {
	gw, _ := Setup()
	if err := gw.Process(); err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
}

func TestParseConfig(t *testing.T) {
	gw, fs := Setup()
	WriteFile(fs, "src/config.yaml", "key: value")
	if err := gw.parseConfig("/home/test/src/config.yaml"); err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}
	if gw.config["key"] != "value" {
		t.Fatalf("Unexpected config value: %v", gw.config["key"])
	}
}

func TestFilesCopiedToBuild(t *testing.T) {
	gw, fs := Setup()
	data1 := "javascript"
	data2 := "css"
	WriteFile(fs, "src/static/js/app.js", data1)
	WriteFile(fs, "src/static/css/app.css", data2)
	gw.Process()
	if s, _ := ReadFile(fs, "build/static/js/app.js"); s != data1 {
		t.Fatalf("Read: %v, Expected: %v", s, data1)
	}
	if s, _ := ReadFile(fs, "build/static/css/app.css"); s != data2 {
		t.Fatalf("Read: %v, Expected: %v", s, data2)
	}
}
