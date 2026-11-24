package gocc

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/carlmjohnson/versioninfo"
	"github.com/mhr3/gocc/internal/cc"
	"github.com/mhr3/gocc/internal/config"
)

// Server translates the C sources that Remote clients send it, with the
// toolchain of the machine it runs on.
//
// It is meant for machines and networks you trust. A client chooses the
// compiler options and the source, and so can make the compiler read and
// write files as the user running the server. Token keeps strangers out; it
// does not make the server safe to expose to them.
type Server struct {
	// Token, when set, has to be presented by every client.
	Token string
	// Timeout is the time one translation may take. Two minutes when zero.
	Timeout time.Duration
	// MaxRequestBytes limits the size of a request. 32 MiB when zero.
	MaxRequestBytes int64
	// Concurrency is the number of translations running at once; further
	// requests wait. The number of CPUs when zero.
	Concurrency int

	once  sync.Once
	slots chan struct{}
}

func (s *Server) timeout() time.Duration {
	if s.Timeout > 0 {
		return s.Timeout
	}
	return 2 * time.Minute
}

// Handler returns the HTTP handler of the server: POST /compile takes a
// CompileRequest and answers with a CompileResult, GET /health describes the
// toolchain.
func (s *Server) Handler() http.Handler {
	s.once.Do(func() {
		concurrency := s.Concurrency
		if concurrency <= 0 {
			concurrency = runtime.NumCPU()
		}
		s.slots = make(chan struct{}, concurrency)
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /compile", s.compile)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	compiler, err := cc.NewCompiler(config.AMD64())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, compileError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"version":  versioninfo.Short(),
		"compiler": compiler.Version(),
		"os":       runtime.GOOS,
	})
}

// requestError is a failure the client caused before anything was compiled.
type requestError struct{ message string }

func (e requestError) Error() string { return e.message }

func badRequest(format string, args ...any) error {
	return requestError{fmt.Sprintf(format, args...)}
}

func (s *Server) compile(w http.ResponseWriter, r *http.Request) {
	if s.Token != "" {
		presented := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(presented), []byte(s.Token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, compileError{Error: "gocc server: missing or wrong token"})
			return
		}
	}

	limit := s.MaxRequestBytes
	if limit <= 0 {
		limit = 32 << 20
	}
	var request CompileRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit)).Decode(&request); err != nil {
		status := http.StatusBadRequest
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeJSON(w, status, compileError{Error: "gocc server: " + err.Error()})
		return
	}

	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-r.Context().Done():
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.timeout())
	defer cancel()
	started := time.Now()
	result, err := translate(ctx, &request)

	status := http.StatusOK
	var invalid requestError
	switch {
	case err == nil:
	case errors.As(err, &invalid):
		status = http.StatusBadRequest
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		status = http.StatusUnprocessableEntity
		err = fmt.Errorf("gocc server: the translation did not finish within %s", s.timeout())
	default:
		status = http.StatusUnprocessableEntity
	}
	log.Printf("%s %s %s: %d in %s", r.RemoteAddr, request.Arch, request.Source, status, time.Since(started).Round(time.Millisecond))

	if err != nil {
		writeJSON(w, status, compileError{Error: err.Error()})
		return
	}
	writeJSON(w, status, result)
}

// What a request may put into a file name, a build tag or the Go source.
var (
	validPackage        = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	validSuffix         = regexp.MustCompile(`^[A-Za-z0-9_.-]*$`)
	validFunctionSuffix = regexp.MustCompile(`^[A-Za-z0-9_]*$`)
	validOS             = regexp.MustCompile(`^[a-z0-9]*$`)
)

// translate runs one translation in a directory of its own, which is gone
// when it returns.
func translate(ctx context.Context, request *CompileRequest) (*CompileResult, error) {
	arch, err := config.For(request.Arch)
	if err != nil {
		return nil, badRequest("%s", err)
	}
	switch {
	case !validPackage.MatchString(request.Package):
		return nil, badRequest("invalid package name %q", request.Package)
	case !validSuffix.MatchString(request.Suffix) || strings.Contains(request.Suffix, ".."):
		return nil, badRequest("invalid suffix %q", request.Suffix)
	case !validFunctionSuffix.MatchString(request.FunctionSuffix):
		return nil, badRequest("invalid function suffix %q", request.FunctionSuffix)
	case !validOS.MatchString(request.OS):
		return nil, badRequest("invalid operating system %q", request.OS)
	}
	if _, ok := request.Files[request.Source]; !ok {
		return nil, badRequest("the source %q is not among the files sent", request.Source)
	}
	if request.OS != "" {
		arch.BuildTags += " && " + request.OS
	}

	tmp, err := os.MkdirTemp("", "gocc-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	// The compiler leaves its own files next to the source, among them an
	// assembly file named like the one to generate.
	sources, output := filepath.Join(tmp, "src"), filepath.Join(tmp, "out")
	if err := os.Mkdir(output, 0o755); err != nil {
		return nil, err
	}
	for name, body := range request.Files {
		local := filepath.FromSlash(name)
		if !filepath.IsLocal(local) {
			return nil, badRequest("invalid file name %q", name)
		}
		path := filepath.Join(sources, local)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return nil, err
		}
	}

	// Paths of this directory mean nothing to the client.
	anonymous := strings.NewReplacer(sources+string(filepath.Separator), "", output+string(filepath.Separator), "")

	source := filepath.Join(sources, filepath.FromSlash(request.Source))
	translator, err := NewLocal(arch, source, output, request.Suffix, request.FunctionSuffix, request.Package, request.Options...)
	if err != nil {
		return nil, err
	}
	translator.WithInternalFunctions = request.WithInternalFunctions
	var diagnostics bytes.Buffer
	translator.Clang.Diagnostics = &diagnostics

	if err := translator.TranslateContext(ctx); err != nil {
		return nil, errors.New(anonymous.Replace(err.Error()))
	}
	result, err := translator.Output()
	if err != nil {
		return nil, err
	}
	result.Diagnostics = anonymous.Replace(diagnostics.String())
	return result, nil
}
