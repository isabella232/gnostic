// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	_ "os"
	"path/filepath"
	"text/template"

	plugins "github.com/googleapis/gnostic/plugins"
)

const newline = "\n"

// ServiceRenderer reads an OpenAPI document and generates code.
type ServiceRenderer struct {
	Templates map[string]*template.Template
	Model     *ServiceModel
}

// NewServiceRenderer creates a renderer.
func NewServiceRenderer(model *ServiceModel) (renderer *ServiceRenderer, err error) {
	renderer = &ServiceRenderer{}
	renderer.Model = model
	// Load templates.
	err = renderer.loadTemplates(templates())
	if err != nil {
		return nil, err
	}
	return renderer, nil
}

// loadTemplates loads templates that will be used by the renderer.
func (renderer *ServiceRenderer) loadTemplates(files map[string]string) (err error) {
	helpers := templateHelpers()
	renderer.Templates = make(map[string]*template.Template, 0)
	for filename, encoding := range files {
		templateData, err := base64.StdEncoding.DecodeString(encoding)
		if err != nil {
			return err
		}
		t, err := template.New(filename).Funcs(helpers).Parse(string(templateData))
		if err != nil {
			return err
		}
		renderer.Templates[filename] = t
	}
	return err
}

// Generate runs the renderer to generate the named files.
func (renderer *ServiceRenderer) Generate(response *plugins.Response, files []string) (err error) {
	for _, filename := range files {
		file := &plugins.File{}
		file.Name = filename
		f := new(bytes.Buffer)

		switch filename {
		case "client.go":
			file.Data, err = renderer.GenerateClient()
		case "types.go":
			file.Data, err = renderer.GenerateTypes()
		case "provider.go":
			file.Data, err = renderer.GenerateProvider()
		case "server.go":
			file.Data, err = renderer.GenerateServer()
		default:
			t := renderer.Templates[filename]
			log.Printf("Generating %s", filename)
			err = t.Execute(f, struct {
				Model *ServiceModel
			}{
				renderer.Model,
			})
			if err != nil {
				response.Errors = append(response.Errors, fmt.Sprintf("ERROR %v", err))
			}
			file.Data = f.Bytes()
		}
		if err != nil {
			response.Errors = append(response.Errors, fmt.Sprintf("ERROR %v", err))
		}
		inputBytes := file.Data
		// run generated Go files through gofmt
		if filepath.Ext(file.Name) == ".go" {
			strippedBytes := stripMarkers(inputBytes)
			file.Data, err = gofmt(file.Name, strippedBytes)
		} else {
			file.Data = inputBytes
		}
		response.Files = append(response.Files, file)
	}
	return
}