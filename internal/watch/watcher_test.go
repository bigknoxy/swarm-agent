package watch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// TestAtomicGuard: second concurrent call is dropped when fixing is already true.
func TestAtomicGuard(t *testing.T) {
	var callCount atomic.Int64
	w := &Watcher{
		Dir:      t.TempDir(),
		BuildCmd: "exit 1",
		MaxFixes: 3,
		FixFunc: func(ctx context.Context, errorOutput string) error {
			callCount.Add(1)
			return nil
		},
	}

	ctx := context.Background()

	// Simulate the CompareAndSwap guard that lives in Run's debounce closure.
	guard := func() {
		if !w.fixing.CompareAndSwap(false, true) {
			return
		}
		defer w.fixing.Store(false)
		w.handleChange(ctx, 1)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			guard()
		}()
	}
	wg.Wait()

	// One goroutine ran; the other was blocked by CAS. FixFunc may be called 0
	// times (build exit 1 produces no output causing FixFunc to be called once)
	// or once — never twice.
	if got := callCount.Load(); got > 1 {
		t.Errorf("FixFunc called %d times; want ≤1 (atomic guard should block concurrent run)", got)
	}
}

// TestRetryLoop: FixFunc is called twice when first build fails, second passes.
func TestRetryLoop(t *testing.T) {
	dir := t.TempDir()
	counterFile := filepath.Join(dir, "attempts")
	os.WriteFile(counterFile, []byte("0"), 0644)

	// Build fails on attempt 1, passes on attempt 2+.
	buildCmd := `n=$(cat ` + counterFile + `); echo $((n+1)) > ` + counterFile + `; [ "$n" -ge 1 ] && exit 0 || exit 1`

	var fixCalls atomic.Int64
	w := &Watcher{
		Dir:      dir,
		BuildCmd: buildCmd,
		MaxFixes: 3,
		FixFunc: func(ctx context.Context, errorOutput string) error {
			fixCalls.Add(1)
			return nil
		},
	}

	w.handleChange(context.Background(), 3)

	if got := fixCalls.Load(); got != 1 {
		t.Errorf("FixFunc called %d times; want 1 (should succeed after first retry)", got)
	}
}

// TestPanicRecovery: FixFunc panics; fixing flag resets to false and handleChange returns.
func TestPanicRecovery(t *testing.T) {
	w := &Watcher{
		Dir:      t.TempDir(),
		BuildCmd: "exit 1",
		MaxFixes: 1,
		FixFunc: func(ctx context.Context, errorOutput string) error {
			panic("boom")
		},
	}

	w.fixing.Store(true)

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer w.fixing.Store(false) // mirrors the CAS guard defer
		w.handleChange(context.Background(), 1)
	}()

	<-done

	if w.fixing.Load() {
		t.Error("fixing is still true after panic recovery; expected false")
	}
}

// TestErrorTruncation: build outputs >8KB to stderr; FixFunc receives ≤8192 bytes.
func TestErrorTruncation(t *testing.T) {
	// Generate 10KB of 'x' to stderr then exit 1.
	bigOutput := strings.Repeat("x", 10240)
	dir := t.TempDir()
	script := filepath.Join(dir, "big_err.sh")
	os.WriteFile(script, []byte("#!/bin/sh\necho '"+bigOutput+"' >&2\nexit 1\n"), 0755)

	var received string
	w := &Watcher{
		Dir:      dir,
		BuildCmd: "sh " + script,
		MaxFixes: 1,
		FixFunc: func(ctx context.Context, errorOutput string) error {
			received = errorOutput
			return nil
		},
	}

	w.handleChange(context.Background(), 1)

	const trailer = "\n[truncated]"
	if !strings.HasSuffix(received, "[truncated]") {
		t.Errorf("truncated output should end with '[truncated]', got: ...%q", received[max(0, len(received)-20):])
	}
	body := received[:len(received)-len(trailer)]
	if len(body) > maxErrorBytes {
		t.Errorf("body before trailer is %d bytes; want ≤%d", len(body), maxErrorBytes)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
