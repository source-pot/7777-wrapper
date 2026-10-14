package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/manifoldco/promptui"
)

func configPaths() []string {
	var paths []string

	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "db-tunnel", "config.toml"),
			filepath.Join(home, ".db-tunnel.toml"),
		)
	}

	exe, err := os.Executable()
	if err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			paths = append(paths, filepath.Join(filepath.Dir(resolved), "config.toml"))
		}
	}

	return paths
}

func findConfig() (string, error) {
	paths := configPaths()
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	msg := "config file not found, checked:"
	for _, p := range paths {
		msg += "\n  " + p
	}
	return "", fmt.Errorf("%s", msg)
}

func run() error {
	cfgPath, err := findConfig()
	if err != nil {
		return err
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	names := make([]string, len(cfg.Databases))
	for i, db := range cfg.Databases {
		names[i] = db.Name
	}

	prompt := promptui.Select{
		Label: "Select a database",
		Items: names,
		Size:  len(names),
	}

	idx, _, err := prompt.Run()
	if err != nil {
		// User pressed Ctrl+C or q
		return nil
	}

	selected := cfg.Databases[idx]

	fmt.Printf("Connecting to %s on port %d...\n", selected.Database, selected.Port)

	args := []string{
		"--port", strconv.Itoa(selected.Port),
		"--database", selected.Database,
	}
	if selected.AWSProfile != "" {
		args = append(args, "--profile", selected.AWSProfile)
	}
	if selected.AWSRegion != "" {
		args = append(args, "--region", selected.AWSRegion)
	}
	if selected.SecurityGroup != "" {
		args = append(args, "--security-group", selected.SecurityGroup)
	}
	if selected.Subnet != "" {
		args = append(args, "--subnet", selected.Subnet)
	}
	if selected.TTL > 0 {
		args = append(args, "--ttl", strconv.Itoa(selected.TTL))
	}
	if selected.Forever != nil && *selected.Forever {
		args = append(args, "--forever")
	}
	if selected.Elasticache {
		args = append(args, "--elasticache")
	}
	if cfg.Verbose {
		args = append(args, "--verbose")
	}
	if cfg.License != "" {
		args = append(args, "--license", cfg.License)
	}

	binary, err := exec.LookPath("7777")
	if err != nil {
		return fmt.Errorf("could not find 7777: %w", err)
	}

	lifetime, limited := tunnelLifetime(selected)
	if !limited {
		return runTunnel(binary, args, 0)
	}

	stopAt := time.Now().Add(lifetime)
	fmt.Printf("Tunnel will be closed at %s, 1 minute before 7777's TTL expires.\n", stopAt.Format("15:04"))
	return runTunnel(binary, args, lifetime)
}

// 7777 shuts its tunnel down after --ttl hours, defaulting to 2.
const defaultTTLHours = 2

// How long before 7777's TTL expires we close the tunnel ourselves.
const shutdownMargin = time.Minute

// tunnelLifetime returns how long db-tunnel lets 7777 run before closing it,
// and false when the tunnel runs forever.
func tunnelLifetime(db DatabaseConfig) (time.Duration, bool) {
	if db.Forever != nil && *db.Forever {
		return 0, false
	}

	ttl := db.TTL
	if ttl <= 0 {
		ttl = defaultTTLHours
	}
	return time.Duration(ttl)*time.Hour - shutdownMargin, true
}

// runTunnel runs 7777 as a child process and, when lifetime is non-zero,
// sends it a single SIGINT once lifetime has passed so it shuts down cleanly.
//
// Ctrl+C and terminal hangups already reach 7777 directly because it shares
// our process group, so we swallow those rather than forwarding them: a second
// SIGINT makes 7777 skip cleaning up its AWS security group rules. SIGTERM is
// only sent to us, so that one is forwarded.
func runTunnel(binary string, args []string, lifetime time.Duration) error {
	cmd := exec.Command(binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start 7777: %w", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var deadline <-chan time.Time
	if lifetime > 0 {
		timer := time.NewTimer(lifetime)
		defer timer.Stop()
		deadline = timer.C
	}

	for {
		select {
		case err := <-done:
			return childResult(err)
		case <-deadline:
			fmt.Println("\nTTL is about to expire, closing the tunnel...")
			cmd.Process.Signal(syscall.SIGINT)
			deadline = nil
		case sig := <-sigCh:
			if sig == syscall.SIGTERM {
				cmd.Process.Signal(sig)
			}
		}
	}
}

// childResult passes 7777's exit code through as our own.
func childResult(err error) error {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return err
	}
	if exitErr.ExitCode() > 0 {
		os.Exit(exitErr.ExitCode())
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
