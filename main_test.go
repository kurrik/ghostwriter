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
	"github.com/kurrik/go-fauxfile"
	"testing"
)

func GetGhostWriter(fs fauxfile.Filesystem) *GhostWriter {
	fs.MkdirAll("/home/test", 0755)
	fs.Chdir("/home/test")
	fs.Mkdir("src", 0755)
	fs.Mkdir("build", 0755)
	c := &Configuration{}
	c.source = "src"
	c.build = "build"
	return &GhostWriter{config: c}
}

func TestFilesystem(t *testing.T) {
	fs := fauxfile.NewMockFilesystem()
	w := GetGhostWriter(fs)
	if err := w.Parse(); err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
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
	return nil
}

func TestParseConfig(t *testing.T) {
	var (
		fs   fauxfile.Filesystem
		conf map[interface{}]interface{}
		err  error
	)
	fs = fauxfile.NewMockFilesystem()
	if err := WriteFile(fs, "config.yaml", "build: build"); err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	if conf, err = ParseConfig(fs, "config.yaml"); err != nil {
		t.Fatalf("Error parsing config: %v", err)
	}
	if conf["build"] != "build" {
		t.Fatalf("Unexpected config build path: %v", conf["build"])
	}
}
