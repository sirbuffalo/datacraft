//go:build js && wasm

package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"sort"
	"syscall/js"

	"mccomp/datapack"
)

type request struct {
	Source      string `json:"source"`
	PackName    string `json:"packName"`
	Description string `json:"description"`
	PackFormat  int    `json:"packFormat"`
}

type response struct {
	OK    bool     `json:"ok"`
	Error string   `json:"error,omitempty"`
	Files []string `json:"files,omitempty"`
	ZIP   string   `json:"zip,omitempty"`
}

func main() {
	compile := js.FuncOf(compile)
	js.Global().Set("mccompCompile", compile)
	select {}
}

func compile(_ js.Value, args []js.Value) any {
	if len(args) != 1 {
		return encode(response{Error: "compiler expected one JSON request"})
	}
	var input request
	if err := json.Unmarshal([]byte(args[0].String()), &input); err != nil {
		return encode(response{Error: "invalid compiler request: " + err.Error()})
	}
	pack, err := datapack.Build(input.Source, datapack.Config{
		PackName: input.PackName, Description: input.Description, PackFormat: input.PackFormat,
	})
	if err != nil {
		return encode(response{Error: err.Error()})
	}
	files := sortedPaths(pack.Files)
	archive, err := makeZIP(pack.Files, files)
	if err != nil {
		return encode(response{Error: "could not create data-pack ZIP: " + err.Error()})
	}
	return encode(response{OK: true, Files: files, ZIP: base64.StdEncoding.EncodeToString(archive)})
}

func sortedPaths(files map[string]string) []string {
	paths := make([]string, 0, len(files))
	for name := range files {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	return paths
}

func makeZIP(files map[string]string, paths []string) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range paths {
		file, err := writer.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := file.Write([]byte(files[name])); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func encode(value response) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
