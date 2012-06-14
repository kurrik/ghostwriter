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
	"fmt"
)

type MockEnvironment struct {
	Files map[string]string
}

func (e *MockEnvironment) GetFiles(path string) {
	for k,v := range e.Files {
		fmt.Printf("%v:%v\n", k, v)
	}
}

func TestEnvironment(t *testing.T) {
	e := MockEnvironment{map[string]string{
		"path": "foobar",
		"path/foo": "foobar baz",
	}}
	e.GetFiles("path")
}
