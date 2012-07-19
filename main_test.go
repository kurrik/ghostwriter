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
	"github.com/kurrik/go-fauxfile"
)

func TestFilesystem(t *testing.T) {
	fs := fauxfile.NewMockFilesystem()
	fs.MkdirAll("/src", 0755)
	fs.MkdirAll("/build", 0755)
	c := &Configuration{}
	c.source = "src"
	c.build = "build"
	w := &GhostWriter{config: c}
	if err := w.Parse(); err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
}
