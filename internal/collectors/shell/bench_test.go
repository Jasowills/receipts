package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// BenchmarkShimOverhead compares a direct no-op exec against the same
// command through the shim. Run with -bench and -benchtime. Hardware and
// Go version print with -v; targets are median under 15ms and p95 under
// 40ms per invocation excluding the command runtime itself.
func BenchmarkShimOverhead(b *testing.B) {
	realDir := b.TempDir()
	runDir := b.TempDir()
	script := filepath.Join(realDir, "noop")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		b.Fatalf("write script: %v", err)
	}
	b.Setenv(EnvRealPath, realDir)
	b.Setenv(EnvShimDir, b.TempDir())
	b.Setenv(EnvRunDir, runDir)
	null, _ := os.Open(os.DevNull)
	const samples = 100
	var shimmed, direct []time.Duration
	for i := 0; i < samples; i++ {
		start := time.Now()
		if code := Main("noop", nil, null, devNull{}, devNull{}); code != 0 {
			b.Fatalf("shim exit = %d", code)
		}
		shimmed = append(shimmed, time.Since(start))
		start = time.Now()
		if err := runDirect(script); err != nil {
			b.Fatalf("direct: %v", err)
		}
		direct = append(direct, time.Since(start))
	}
	smed, sp95 := quantiles(shimmed)
	dmed, _ := quantiles(direct)
	overhead := smed - dmed
	b.ReportMetric(float64(smed.Nanoseconds()), "shim-median-ns/op")
	b.ReportMetric(float64(sp95.Nanoseconds()), "shim-p95-ns/op")
	b.ReportMetric(float64(overhead.Nanoseconds()), "overhead-median-ns/op")
	b.Logf("shim median=%v p95=%v direct median=%v overhead median=%v", smed, sp95, dmed, overhead)
}

func quantiles(ds []time.Duration) (med, p95 time.Duration) {
	cp := append([]time.Duration{}, ds...)
	sort.Slice(cp, func(a, b int) bool { return cp[a] < cp[b] })
	if len(cp) == 0 {
		return 0, 0
	}
	med = cp[len(cp)/2]
	idx := int(float64(len(cp)) * 0.95)
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return med, cp[idx]
}

type devNull struct{}

func (devNull) Write(p []byte) (int, error) { return len(p), nil }

func runDirect(script string) error {
	cmd := exec.Command(script)
	cmd.Stdout = devNull{}
	cmd.Stderr = devNull{}
	return cmd.Run()
}
