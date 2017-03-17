// Copyright 2017 Arne Roomann-Kurrik
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
	"github.com/kurrik/fauxfile"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type ImageMeta struct {
	Width  int
	Height int
	Path   string
	Data   map[string]string
}

func NewImageMeta(fs fauxfile.Filesystem, path string, data map[string]string) (meta ImageMeta, err error) {
	var (
		img  image.Image
		file fauxfile.File
	)
	if file, err = fs.Open(path); err != nil {
		return
	}
	defer file.Close()
	if img, _, err = image.Decode(file); err != nil {
		return
	}
	meta = ImageMeta{
		Width:  img.Bounds().Dx(),
		Height: img.Bounds().Dy(),
		Path:   path,
		Data:   data,
	}
	return
}
