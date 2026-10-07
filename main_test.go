package main

import (
	"os"
	"slices"
	"testing"

	"github.com/fredrikaverpil/claudeline/internal/creds"
	"github.com/fredrikaverpil/claudeline/internal/paths"
)

func TestKeychainServiceNameWiring(t *testing.T) {
	t.Parallel()

	configDir := "/Users/oa/.claude-work"
	suffix := paths.ConfigDirSuffix(configDir)
	if got := creds.KeychainServiceName(configDir); got != "Claude Code-credentials"+suffix {
		t.Errorf("KeychainServiceName() = %q, want suffix %q", got, suffix)
	}
}

func TestDefaultSegments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       config
		loginType string
		want      []string
	}{
		{
			name: "default",
			want: []string{"plan", "model", "context", "5h", "7d", "extra", "status", "update"},
		},
		{
			name: "legacy flags",
			cfg:  config{showCwd: true, showGitBranch: true, showCost: true},
			want: []string{
				"plan",
				"model",
				"cwd",
				"branch",
				"context",
				"5h",
				"7d",
				"cost",
				"extra",
				"status",
				"update",
			},
		},
		{
			name:      "cost always shown for API users",
			loginType: creds.ProviderAPI,
			want:      []string{"plan", "model", "context", "5h", "7d", "cost", "extra", "status", "update"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := defaultSegments(tt.cfg, tt.loginType)
			if !slices.Equal(got, tt.want) {
				t.Errorf("defaultSegments() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkRun(b *testing.B) {
	// Offline testdata. Keep the Pro payload: it has rate_limits, pr and
	// prompt_cache, so the full render path runs.
	stdinFile := "internal/stdin/testdata/stdin_pro_sonnet.json"
	usageFile := "internal/usage/testdata/usage_pro.json"
	statusFile := "internal/status/testdata/status.json"

	stdinData, err := os.ReadFile(stdinFile)
	if err != nil {
		b.Fatalf("read stdin testdata: %v", err)
	}

	cfg := config{
		usageFile:       usageFile,
		statusFile:      statusFile,
		gitBranchMaxLen: 30,
		cwdMaxLen:       30,
	}

	// Discard stdout to avoid benchmark noise.
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer devNull.Close()
	origStdout := os.Stdout
	os.Stdout = devNull
	defer func() { os.Stdout = origStdout }()

	b.ResetTimer()
	for b.Loop() {
		r, w, err := os.Pipe()
		if err != nil {
			b.Fatal(err)
		}
		if _, err := w.Write(stdinData); err != nil {
			b.Fatal(err)
		}
		w.Close()

		os.Stdin = r
		if err := run(cfg); err != nil {
			b.Fatalf("run: %v", err)
		}
		r.Close()
	}
}
