package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/manifoldco/promptui"
)

func configPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("could not resolve executable symlink: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), "config.toml"), nil
}

func run() error {
	cfgPath, err := configPath()
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

	cmd := exec.Command("7777", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start 7777: %w", err)
	}

	// Forward signals to child process
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	done := make(chan struct{})
	go func() {
		select {
		case sig := <-sigCh:
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		case <-done:
		}
	}()

	err = cmd.Wait()
	close(done)
	return err
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
