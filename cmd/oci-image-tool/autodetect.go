// Copyright 2016 The Linux Foundation
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
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/opencontainers/image-spec/schema"
	"github.com/pkg/errors"
)

// supported autodetection types
const (
	typeImageLayout  = "imageLayout"
	typeImage        = "image"
	typeManifest     = "manifest"
	typeManifestList = "manifestList"
	typeConfig       = "config"
)

// autodetect detects the validation type for the given path
// or an error if the validation type could not be resolved.
func autodetect(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", errors.Wrapf(err, "unable to access path") // err from os.Stat includes path name
	}

	if fi.IsDir() {
		return typeImageLayout, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", errors.Wrap(err, "unable to open file") // os.Open includes the filename
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(io.LimitReader(f, 512)) // read some initial bytes to detect content
	if err != nil {
		return "", errors.Wrap(err, "unable to read")
	}

	mimeType := http.DetectContentType(buf)

	switch mimeType {
	case "application/x-gzip":
		return typeImage, nil

	case "application/octet-stream":
		return typeImage, nil

	case "text/plain; charset=utf-8":
		// might be a JSON file, will be handled below

	default:
		return "", errors.New("unknown file type")
	}

	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return "", errors.Wrap(err, "unable to seek")
	}

	header := struct {
		SchemaVersion int         `json:"schemaVersion"`
		MediaType     string      `json:"mediaType"`
		Config        interface{} `json:"config"`
	}{}

	if err := json.NewDecoder(f).Decode(&header); err != nil {
		return "", errors.Wrap(err, "unable to parse JSON")
	}

	switch {
	case header.MediaType == string(schema.MediaTypeManifest):
		return typeManifest, nil

	case header.MediaType == string(schema.MediaTypeManifestList):
		return typeManifestList, nil

	case header.MediaType == "" && header.SchemaVersion == 0 && header.Config != nil:
		// config files don't have mediaType/schemaVersion header
		return typeConfig, nil
	}

	return "", errors.New("unknown media type")
}
