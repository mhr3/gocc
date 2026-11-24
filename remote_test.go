package gocc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mhr3/gocc/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startServer runs a Server for the duration of a test. The tests translate
// for the architecture of the machine, the one whose headers are sure to be
// installed.
func startServer(t *testing.T, server *Server) string {
	t.Helper()
	if _, err := config.FindClang(); err != nil {
		t.Skip("clang is not installed")
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	return httpServer.URL
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	}
}

func readDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	files := map[string]string{}
	for _, entry := range entries {
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, err)
		files[entry.Name()] = string(body)
	}
	return files
}

const validSource = `#include <stdint.h>

// gocc: Sum(p *int64, n int64) int64
int64_t sum(const int64_t *p, int64_t n) {
    int64_t s = 0;
    for (int64_t i = 0; i < n; i++) s += p[i];
    return s;
}
`

// A translation through a server has to give the files the same translation
// gives on the spot, whatever is asked for.
func TestRemoteMatchesLocal(t *testing.T) {
	url := startServer(t, &Server{})

	// a source in a subdirectory that includes a file next to it, one in
	// another directory, and one that only the compiler can find
	tree := t.TempDir()
	writeFiles(t, tree, map[string]string{
		"kernels/csrc/weighted.c": `#include <stdint.h>
#include "scale.h"
#  include "../../shared/bias.h"

// gocc: Weighted(p *int64, n int64) int64
int64_t weighted(const int64_t *p, int64_t n) {
    int64_t s = BIAS;
    for (int64_t i = 0; i < n; i++) s += scale(p[i]);
    return s;
}
`,
		"kernels/csrc/scale.h":  "#include \"factor.h\"\nstatic inline int64_t scale(int64_t x) { return x * FACTOR; }\n",
		"kernels/csrc/factor.h": "#define FACTOR 3\n",
		"shared/bias.h":         "#define BIAS 7\n",
	})

	for _, tc := range []struct {
		name                  string
		source                string
		pkg, suffix, fnSuffix string
		osTag                 string
		internal              bool
		options               []string
		files                 []string // the names of the generated files
	}{
		{
			name:    "plain",
			source:  "internal/asm/generated/testdata/test_src.c",
			pkg:     "generated",
			options: []string{"-O3"},
			files:   []string{"test_src.go", "test_src.s"},
		},
		{
			name:     "every option",
			source:   "internal/asm/generated/testdata/test_calls.c",
			suffix:   "_" + runtime.GOARCH,
			fnSuffix: "_remote",
			osTag:    "linux",
			internal: true,
			options:  []string{"-O2"},
			files:    []string{"test_calls_" + runtime.GOARCH + ".go", "test_calls_" + runtime.GOARCH + ".s"},
		},
		{
			name:    "included files",
			source:  filepath.Join(tree, "kernels", "csrc", "weighted.c"),
			pkg:     "kernels",
			options: []string{"-O3"},
			files:   []string{"weighted.go", "weighted.s"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// directories of one name, which is the package when none is given
			localDir := filepath.Join(t.TempDir(), "translated")
			remoteDir := filepath.Join(t.TempDir(), "translated")
			require.NoError(t, os.Mkdir(localDir, 0o755))

			// the translator leaves files next to the source it is given
			source := filepath.Join(t.TempDir(), filepath.Base(tc.source))
			body, err := os.ReadFile(tc.source)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(source, body, 0o644))
			if filepath.IsAbs(tc.source) {
				source = tc.source
			}

			arch, err := config.For(runtime.GOARCH)
			require.NoError(t, err)
			if tc.osTag != "" {
				arch.BuildTags += " && " + tc.osTag
			}
			local, err := NewLocal(arch, source, localDir, tc.suffix, tc.fnSuffix, tc.pkg, tc.options...)
			require.NoError(t, err)
			local.WithInternalFunctions = tc.internal
			require.NoError(t, local.Translate())

			remote := Remote{
				Server: url, Arch: runtime.GOARCH, Source: source, Output: remoteDir,
				Package: tc.pkg, Suffix: tc.suffix, FuncSuffix: tc.fnSuffix, OS: tc.osTag,
				Options: tc.options, WithInternalFunctions: tc.internal,
			}
			require.NoError(t, remote.Translate())

			want := readDir(t, localDir)
			got := readDir(t, remoteDir)
			require.Len(t, want, 2)
			for _, name := range tc.files {
				require.Contains(t, got, name)
				assert.Equal(t, want[name], got[name], name)
			}
			assert.Len(t, got, 2)
		})
	}
}

func TestRemoteReportsFailures(t *testing.T) {
	url := startServer(t, &Server{Token: "secret"})

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"valid.c": validSource,
		"broken.c": `#include <stdint.h>

// gocc: Broken(a int64) int64
int64_t broken(int64_t a) {
    return undeclared + a;
}
`,
	})
	remote := func(source, arch, token string) (Remote, string) {
		output := filepath.Join(t.TempDir(), "out")
		return Remote{Server: url, Token: token, Arch: arch, Source: filepath.Join(dir, source), Output: output, Package: "p"}, output
	}

	t.Run("compiler error", func(t *testing.T) {
		r, output := remote("broken.c", runtime.GOARCH, "secret")
		err := r.Translate()
		// the diagnostic as the compiler gives it, with the name of the
		// source in place of where the server kept it
		require.ErrorContains(t, err, "broken.c:5:12: error: use of undeclared identifier 'undeclared'")
		assert.NotContains(t, err.Error(), os.TempDir())
		assert.NoDirExists(t, output, "nothing is written when the translation fails")
	})
	t.Run("unknown architecture", func(t *testing.T) {
		r, _ := remote("valid.c", "riscv", "secret")
		require.ErrorContains(t, r.Translate(), "unsupported architecture: riscv")
	})
	t.Run("token", func(t *testing.T) {
		r, _ := remote("valid.c", runtime.GOARCH, "")
		require.ErrorContains(t, r.Translate(), "missing or wrong token")
		r, _ = remote("valid.c", runtime.GOARCH, "guess")
		require.ErrorContains(t, r.Translate(), "missing or wrong token")

		r, output := remote("valid.c", runtime.GOARCH, "secret")
		require.NoError(t, r.Translate())
		assert.Len(t, readDir(t, output), 2)
	})
	t.Run("no server", func(t *testing.T) {
		r, _ := remote("valid.c", runtime.GOARCH, "secret")
		r.Server = "localhost:1"
		require.ErrorContains(t, r.Translate(), "gocc server")
	})
}

// post sends a request as it is, which a Remote would not always do.
func post(t *testing.T, url string, request CompileRequest) (int, CompileResult, string) {
	t.Helper()
	body, err := json.Marshal(request)
	require.NoError(t, err)
	response, err := http.Post(url+"/compile", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer response.Body.Close()

	var result CompileResult
	var failure compileError
	if response.StatusCode == http.StatusOK {
		require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
	} else {
		require.NoError(t, json.NewDecoder(response.Body).Decode(&failure))
	}
	return response.StatusCode, result, failure.Error
}

func TestServerRejectsWhatCouldLeaveItsDirectory(t *testing.T) {
	url := startServer(t, &Server{})
	valid := func() CompileRequest {
		return CompileRequest{
			Arch: runtime.GOARCH, Package: "p", Source: "valid.c",
			Files: map[string][]byte{"valid.c": []byte(validSource)},
		}
	}

	status, result, _ := post(t, url, valid())
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "valid.s", result.Asm.Name)
	assert.Equal(t, "valid.go", result.Go.Name)

	for name, change := range map[string]func(*CompileRequest){
		"file outside":      func(r *CompileRequest) { r.Files["../escaped.h"] = []byte("int x;") },
		"absolute file":     func(r *CompileRequest) { r.Files["/tmp/escaped.h"] = []byte("int x;") },
		"suffix with a dir": func(r *CompileRequest) { r.Suffix = "/../../escaped" },
		"source not sent":   func(r *CompileRequest) { r.Source = "other.c" },
		"package":           func(r *CompileRequest) { r.Package = "p\nimport \"os\"" },
		"function suffix":   func(r *CompileRequest) { r.FunctionSuffix = "() {}\nfunc x" },
		"operating system":  func(r *CompileRequest) { r.OS = "linux\n//go:build ignore" },
	} {
		t.Run(name, func(t *testing.T) {
			request := valid()
			change(&request)
			status, _, message := post(t, url, request)
			assert.Equal(t, http.StatusBadRequest, status, message)
		})
	}
}

func TestServerLimits(t *testing.T) {
	request := CompileRequest{
		Arch: runtime.GOARCH, Package: "p", Source: "valid.c",
		Files: map[string][]byte{"valid.c": []byte(validSource)},
	}

	t.Run("request size", func(t *testing.T) {
		url := startServer(t, &Server{MaxRequestBytes: 64})
		status, _, message := post(t, url, request)
		assert.Equal(t, http.StatusRequestEntityTooLarge, status, message)
	})
	t.Run("time", func(t *testing.T) {
		// A few lines that expand to 262144 statements, which take the
		// compiler minutes: the answer can only come in time if the
		// compiler is stopped.
		slow := CompileRequest{
			Arch: runtime.GOARCH, Package: "p", Source: "slow.c", Options: []string{"-O3"},
			Files: map[string][]byte{"slow.c": []byte(`#include <stdint.h>
#define A(x) x x x x x x x x
#define B(x) A(A(A(x)))
#define C(x) B(B(x))

// gocc: Slow(a int64) int64
int64_t slow(int64_t a) {
    volatile int64_t s = a;
    C(s = s * 31 + (s >> 3);)
    return s;
}
`)},
		}
		url := startServer(t, &Server{Timeout: 500 * time.Millisecond})
		started := time.Now()
		status, _, message := post(t, url, slow)
		assert.Equal(t, http.StatusUnprocessableEntity, status)
		assert.Contains(t, message, "did not finish within 500ms")
		assert.Less(t, time.Since(started), 20*time.Second)
	})
}

func TestServerReturnsCompilerWarnings(t *testing.T) {
	url := startServer(t, &Server{})
	status, result, message := post(t, url, CompileRequest{
		Arch: runtime.GOARCH, Package: "p", Source: "warn.c",
		Files: map[string][]byte{"warn.c": []byte(`#include <stdint.h>

// gocc: Warn(a int64) int64
int64_t warn(int64_t a) {
    a + 1;
    return a;
}
`)},
	})
	require.Equal(t, http.StatusOK, status, message)
	assert.Contains(t, result.Diagnostics, "warn.c:5:7: warning")
	assert.NotContains(t, result.Diagnostics, os.TempDir())
}
