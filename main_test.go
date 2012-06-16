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
	"testing"
	"strings"
	"sort"
	"path/filepath"
)

type MockEnvironment struct {
	Files map[string]string
}

func (e *MockEnvironment) GetFiles(path string) []string {
	paths := make([]string, 0)
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path = path + string(filepath.Separator)
	}
	for p, _ := range e.Files {
		if strings.HasPrefix(p, path) {
			p = strings.Replace(p, path, "", -1)
			parts := strings.Split(p, string(filepath.Separator))
			if len(parts) == 1 {
				paths = append(paths, parts[0])
			}
		}
	}
	return paths
}

func (e *MockEnvironment) GetDirs(path string) []string {
	dirmap := map[string]bool{}
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path = path + string(filepath.Separator)
	}
	for p, _ := range e.Files {
		if strings.HasPrefix(p, path) {
			p = strings.Replace(p, path, "", -1)
			parts := strings.Split(p, string(filepath.Separator))
			if len(parts) > 1 {
				dirmap[parts[0]] = true
			}
		}
	}
	dirs := make([]string, len(dirmap))
	i := 0
	for dir, _ := range(dirmap) {
		dirs[i] = dir
		i++
	}
	return dirs
}

func Expect(t *testing.T, exp interface{}, act interface{}) {
	if exp != act {
		t.Fatalf("Expected %v, got %v", exp, act)
	}
}

func ExpectStringSet(t *testing.T, exp sort.StringSlice, act sort.StringSlice) {
	if len(exp) != len(act) {
		t.Fatalf("Expected length %v, got %v", len(exp), len(act))
	}
	exp.Sort()
	act.Sort()
	for i, _ := range exp {
		if exp[i] != act[i] {
			t.Fatalf("Expected %v at %v, got %v", exp[i], i, act[i])
		}
	}
}

func TestEnvironment(t *testing.T) {
	e := MockEnvironment{map[string]string{
		"path/a.txt": "Contents of a.txt",
		"path/b.txt": "Contents of b.txt",
		"path/dir1/c.txt": "Contents of c.txt",
		"path/dir2/d.txt": "Contents of d.txt",
	}}
	paths := e.GetFiles("path")
	ExpectStringSet(t, []string{"a.txt", "b.txt"}, paths)
	paths = e.GetDirs("path")
	ExpectStringSet(t, []string{"dir1", "dir2"}, paths)
}
