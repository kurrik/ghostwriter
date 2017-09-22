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
	"fmt"
	"github.com/kurrik/fauxfile"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
)

type ImageData struct {
	Width  int
	Height int
	Path   string
}

func NewImageData(fs fauxfile.Filesystem, srcPath string, dstPath string) (data ImageData, err error) {
	var (
		img  image.Image
		file fauxfile.File
	)
	if file, err = fs.Open(srcPath); err != nil {
		return
	}
	defer file.Close()
	if img, _, err = image.Decode(file); err != nil {
		return
	}
	data = ImageData{
		Width:  img.Bounds().Dx(),
		Height: img.Bounds().Dy(),
		Path:   dstPath,
	}
	return
}

type Image struct {
	meta     ImageMeta
	data     ImageData
	variants map[string]ImageData
}

func NewImage(gw *GhostWriter, meta ImageMeta, postSrcDir string, postDstDir string) (out *Image, err error) {
	out = &Image{
		meta:     meta,
		variants: map[string]ImageData{},
	}
	var (
		srcPath string = filepath.Join(postSrcDir, meta.Src)
		dstPath string = filepath.Join(postDstDir, meta.Src)
	)
	if out.data, err = NewImageData(gw.fs, srcPath, dstPath); err != nil {
		return
	}
	for key, variantMeta := range meta.Variants {
		if variantMeta.Src != nil {
			srcPath = filepath.Join(postSrcDir, *variantMeta.Src)
			dstPath = filepath.Join(postDstDir, *variantMeta.Src)
			if out.variants[key], err = NewImageData(gw.fs, srcPath, dstPath); err != nil {
				return
			}
		}
	}
	return
}

func (i *Image) Data() ImageData {
	return i.data
}

func (i *Image) Variants() map[string]ImageData {
	return i.variants
}

func (i *Image) Variant(key string) (out ImageData, err error) {
	var exists bool
	if out, exists = i.variants[key]; !exists {
		err = fmt.Errorf("Could not get image variant with key %v", key)
		return
	}
	return
}

func (i *Image) HasMetadata(key string) (exists bool) {
	_, exists = i.meta.Metadata[key]
	return
}

func (i *Image) Metadata() map[string]string {
	return i.meta.Metadata
}
