package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestTunnelLifetime(t *testing.T) {
	yes, no := true, false

	tests := []struct {
		name        string
		db          DatabaseConfig
		wantLimited bool
		want        time.Duration
	}{
		{"default ttl", DatabaseConfig{}, true, 2*time.Hour - time.Minute},
		{"explicit ttl", DatabaseConfig{TTL: 5}, true, 5*time.Hour - time.Minute},
		{"forever false", DatabaseConfig{TTL: 1, Forever: &no}, true, time.Hour - time.Minute},
		{"forever", DatabaseConfig{TTL: 1, Forever: &yes}, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, limited := tunnelLifetime(tt.db)
			if limited != tt.wantLimited || got != tt.want {
				t.Errorf("got (%v, %v), want (%v, %v)", got, limited, tt.want, tt.wantLimited)
			}
		})
	}
}

// fake7777 writes a script that records each SIGINT it receives and exits
// cleanly shortly after the first, like 7777 does after tearing down its
// tunnel.
func fake7777(t *testing.T) (binary, logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "signals.log")
	binary = filepath.Join(dir, "7777")
	script := `#!/bin/sh
stopping=0
trap 'echo INT >> "` + logPath + `"; stopping=1' INT
ticks=0
while [ $ticks -lt 10 ]; do
	sleep 0.05
	[ $stopping -eq 1 ] && ticks=$((ticks + 1))
done
`
	if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
		t.Fatalf("could not write fake 7777: %v", err)
	}
	return binary, logPath
}

func TestRunTunnel_SendsOneSIGINTAtDeadline(t *testing.T) {
	binary, logPath := fake7777(t)

	start := time.Now()
	if err := runTunnel(binary, nil, 300*time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 300*time.Millisecond {
		t.Errorf("tunnel closed after %v, before the deadline", elapsed)
	}

	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("fake 7777 never received SIGINT: %v", err)
	}
	if got := strings.Count(string(log), "INT"); got != 1 {
		t.Errorf("fake 7777 received %d SIGINTs, want exactly 1", got)
	}
}

func TestRunTunnel_ChildExitsBeforeDeadline(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "7777")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := runTunnel(binary, nil, time.Hour); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Ctrl+C in a terminal signals the whole foreground process group. db-tunnel
// must survive it and wait for 7777 to finish its own cleanup and exit.
func TestRunTunnel_CtrlCWaitsForChild(t *testing.T) {
	if binary := os.Getenv("DB_TUNNEL_HELPER_BINARY"); binary != "" {
		runTunnel(binary, nil, time.Hour)
		return
	}

	binary, logPath := fake7777(t)
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunTunnel_CtrlCWaitsForChild$")
	cmd.Env = append(os.Environ(), "DB_TUNNEL_HELPER_BINARY="+binary)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("db-tunnel did not exit cleanly: %v", err)
	}

	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("fake 7777 never received SIGINT: %v", err)
	}
	if got := strings.Count(string(log), "INT"); got != 1 {
		t.Errorf("fake 7777 received %d SIGINTs, want exactly 1", got)
	}
}
