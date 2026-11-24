package gocc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CompileRequest is what a Remote sends to a Server: a C source, the files it
// includes and the options of the translation.
type CompileRequest struct {
	Arch    string   `json:"arch"`
	Package string   `json:"package"`
	Options []string `json:"options,omitempty"`
	// Suffix is added to the names of the generated files, FunctionSuffix to
	// the names of the generated functions.
	Suffix         string `json:"suffix,omitempty"`
	FunctionSuffix string `json:"functionSuffix,omitempty"`
	// OS, when set, is added to the build tags of the generated files. It is
	// the operating system of the client: the server may run another one.
	OS                    string `json:"os,omitempty"`
	WithInternalFunctions bool   `json:"withInternalFunctions,omitempty"`
	// Source names the file to translate among Files.
	Source string `json:"source"`
	// Files holds the source and the files it includes, by their
	// slash-separated path relative to a common directory.
	Files map[string][]byte `json:"files"`
}

// CompileResult is the answer of a Server to a CompileRequest.
type CompileResult struct {
	Asm File `json:"asm"`
	Go  File `json:"go"`
	// Diagnostics is what the compiler printed while succeeding: its warnings.
	Diagnostics string `json:"diagnostics,omitempty"`
}

// File is a generated file, named without a directory.
type File struct {
	Name string `json:"name"`
	Body []byte `json:"body"`
}

// compileError is the body of every answer of a Server other than 200.
type compileError struct {
	Error string `json:"error"`
}

// Remote translates a C file to Go assembly on another machine, which runs a
// Server. The machine running the Remote needs no C toolchain.
type Remote struct {
	// Server is the address of the server, such as http://mac.local:8080.
	Server string
	// Token is the secret the server was started with, if any.
	Token string

	Arch                  string
	Source                string
	Output                string // directory the generated files are written to
	Package               string // name of that directory when empty
	Suffix                string
	FuncSuffix            string
	OS                    string
	Options               []string
	WithInternalFunctions bool

	// Client is used for the request; one with a timeout longer than a
	// compilation should take when nil.
	Client *http.Client
}

// localInclude matches an #include of a file named in quotes, which the
// compiler looks for next to the including file first.
var localInclude = regexp.MustCompile(`(?m)^[ \t]*#[ \t]*include[ \t]+"([^"\n]+)"`)

// collectSources reads a C source and the files it includes with quotes,
// directly or not, as far as they exist relative to the file including them.
// The server has no other way to see them. It returns the files by their
// slash-separated path relative to the directory containing all of them, and
// the path of the source itself.
func collectSources(source string) (files map[string][]byte, main string, err error) {
	source, err = filepath.Abs(source)
	if err != nil {
		return nil, "", err
	}

	contents := map[string][]byte{}
	pending := []string{source}
	for len(pending) > 0 {
		path := pending[0]
		pending = pending[1:]
		if _, seen := contents[path]; seen {
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, "", err
		}
		contents[path] = body

		for _, match := range localInclude.FindAllSubmatch(body, -1) {
			included := filepath.Join(filepath.Dir(path), filepath.FromSlash(string(match[1])))
			// Anything else is for the include paths of the server's compiler.
			if info, err := os.Stat(included); err == nil && info.Mode().IsRegular() {
				pending = append(pending, included)
			}
		}
	}

	// The deepest directory that contains every file.
	root := filepath.Dir(source)
	for path := range contents {
		for {
			rel, err := filepath.Rel(root, path)
			if err == nil && filepath.IsLocal(rel) {
				break
			}
			parent := filepath.Dir(root)
			if parent == root {
				return nil, "", fmt.Errorf("%s and %s have no directory in common", source, path)
			}
			root = parent
		}
	}

	files = make(map[string][]byte, len(contents))
	for path, body := range contents {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, "", err
		}
		files[filepath.ToSlash(rel)] = body
		if path == source {
			main = filepath.ToSlash(rel)
		}
	}
	return files, main, nil
}

func (t *Remote) endpoint(path string) string {
	server := strings.TrimRight(t.Server, "/")
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}
	return server + path
}

// Translate has the server translate the source file and writes the
// generated files to the output directory.
func (t *Remote) Translate() error {
	files, source, err := collectSources(t.Source)
	if err != nil {
		return err
	}
	pkg := t.Package
	if pkg == "" {
		pkg = packageOf(t.Output)
	}
	body, err := json.Marshal(CompileRequest{
		Arch:                  t.Arch,
		Package:               pkg,
		Options:               t.Options,
		Suffix:                t.Suffix,
		FunctionSuffix:        t.FuncSuffix,
		OS:                    t.OS,
		WithInternalFunctions: t.WithInternalFunctions,
		Source:                source,
		Files:                 files,
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, t.endpoint("/compile"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if t.Token != "" {
		request.Header.Set("Authorization", "Bearer "+t.Token)
	}
	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("gocc server: %w", err)
	}
	defer response.Body.Close()
	answer, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("gocc server %s: %w", t.Server, err)
	}

	if response.StatusCode != http.StatusOK {
		var failure compileError
		if json.Unmarshal(answer, &failure) == nil && failure.Error != "" {
			// what the translation would have printed here
			return errors.New(strings.TrimRight(failure.Error, "\n"))
		}
		return fmt.Errorf("gocc server %s: %s", t.Server, response.Status)
	}

	var result CompileResult
	if err := json.Unmarshal(answer, &result); err != nil {
		return fmt.Errorf("gocc server %s: %w", t.Server, err)
	}
	if result.Diagnostics != "" {
		fmt.Fprint(os.Stderr, result.Diagnostics)
	}
	if err := os.MkdirAll(t.Output, 0o755); err != nil {
		return err
	}
	for _, file := range []File{result.Asm, result.Go} {
		name := filepath.Base(file.Name)
		if file.Name == "" || name == "." || name == string(filepath.Separator) {
			return fmt.Errorf("gocc server %s: the answer lacks a file", t.Server)
		}
		if err := os.WriteFile(filepath.Join(t.Output, name), file.Body, 0o644); err != nil {
			return err
		}
	}
	return nil
}
