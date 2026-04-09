package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

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

	return syscall.Exec(binary, append([]string{"7777"}, args...), os.Environ())
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
