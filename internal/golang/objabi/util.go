// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package objabi

import (
	"fmt"

	"runtime"
)

const (
	ElfRelocOffset   = 256
	MachoRelocOffset = 2048    // reserve enough space for ELF relocations
	GlobalDictPrefix = ".dict" // prefix for names of global dictionaries
)

// HeaderString returns the toolchain configuration string written in
// Go object headers. This string ensures we don't attempt to import
// or link object files that are incompatible with each other. This
// string always starts with "go object ".
func HeaderString() string {
	archExtra := ""
	/*
		if k, v := runtime.GOGOARCH(); k != "" && v != "" {
			archExtra = " " + k + "=" + v
		}
	*/
	return fmt.Sprintf("go object %s %s %s%s\n",
		runtime.GOOS, runtime.GOARCH,
		runtime.Version(), archExtra,
		//strings.Join(runtime.Experiment.Enabled(), ","),
	)
}
