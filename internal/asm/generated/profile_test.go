package generated_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mhr3/gocc/internal/asm/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A CPU profile interrupts the program wherever it is, and the runtime then
// has to unwind the stack from that instruction. The C helpers run with the
// stack pointer of the function that called them and keep the return address
// where the compiler pleases, so this is where a translation shows whether
// the runtime can make sense of them.
//
// Test_calls_nested spends its time in a helper that calls another helper.
// On arm64 that used to end the program with "traceback stuck": between the
// inner call and the next write of the link register, the register holds an
// address in the helper itself.
func TestCallsUnwindUnderCPUProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.pprof")
	out, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, pprof.StartCPUProfile(out))

	p := make([]int64, 64)
	for i := range p {
		p[i] = int64(i*i + 1)
	}
	want := generated.Test_calls_nested(p, 3)

	var wg sync.WaitGroup
	deadline := time.Now().Add(3 * time.Second)
	for range 4 {
		wg.Go(func() {
			for time.Now().Before(deadline) {
				for range 200 {
					if got := generated.Test_calls_nested(p, 3); got != want {
						t.Errorf("got %d, want %d", got, want)
						return
					}
				}
			}
		})
	}
	wg.Wait()
	pprof.StopCPUProfile()
	require.NoError(t, out.Close())

	// Every sample taken in the translated code has to lead back here.
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("the go tool is needed to read the profile")
	}
	traces, err := exec.Command(goTool, "tool", "pprof", "-traces", path).Output()
	require.NoError(t, err)

	translated := map[string]bool{"Test_calls_nested": true, "mix_with_locals": true, "weighted_sum": true}
	var samples, broken int
	for _, sample := range strings.Split(string(traces), "-----------+")[1:] {
		var stack []string
		for _, line := range strings.Split(sample, "\n")[1:] {
			if fields := strings.Fields(line); len(fields) > 0 {
				stack = append(stack, fields[len(fields)-1])
			}
		}
		if len(stack) == 0 {
			continue
		}
		innermost := stack[0][strings.LastIndex(stack[0], ".")+1:]
		if !translated[innermost] {
			continue
		}
		samples++
		if !strings.Contains(strings.Join(stack, "\n"), t.Name()) {
			broken++
			if broken <= 3 {
				t.Errorf("a stack that does not reach the test:\n\t%s", strings.Join(stack, "\n\t"))
			}
		}
	}
	if samples == 0 {
		t.Skip("the profile has no sample in the translated code")
	}
	assert.Zero(t, broken, "of %d distinct stacks", samples)
	t.Logf("%d distinct stacks in the translated code, all complete", samples-broken)
}
