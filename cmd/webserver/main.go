// The webserver translates C sources for gocc clients that have no C
// toolchain of their own, or that should all use the compiler of one machine.
//
//	webserver --listen :8080
//	gocc --server http://that-machine:8080 file.c -o . -a arm64 -O3
//
// See gocc.Server for who may be allowed to reach it.
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mhr3/gocc"
)

// loopback reports whether an address to listen on is only reachable from
// this machine.
func loopback(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func main() {
	listen := flag.String("listen", "localhost:8080", "address to listen on; :8080 accepts connections from other machines")
	timeout := flag.Duration("timeout", 2*time.Minute, "time one translation may take")
	concurrency := flag.Int("concurrency", 0, "translations running at once (default: the number of CPUs)")
	flag.Parse()

	server := &gocc.Server{
		Token:       os.Getenv("GOCC_TOKEN"),
		Timeout:     *timeout,
		Concurrency: *concurrency,
	}
	if server.Token == "" && !loopback(*listen) {
		log.Printf("warning: anyone who can reach %s can compile here, and with that read and write files as this user; set GOCC_TOKEN to require a token", *listen)
	}

	log.Printf("listening on %s", *listen)
	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(httpServer.ListenAndServe())
}
