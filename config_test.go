package main

import (
	"os"
	"path/filepath"
	"testing"
)


func writeTestConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("could not write test config: %v", err)
	}
	return path
}

func TestLoadConfig_FullConfig(t *testing.T) {
	configPath := writeTestConfig(t, t.TempDir(), `
aws_profile = "default"
aws_region = "ap-southeast-2"

[[database]]
name = "Prod Main"
database = "prod-master"
port = 7700

[[database]]
name = "Stage EU"
database = "stage-eu"
port = 7710
aws_profile = "other"
aws_region = "eu-west-1"
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AWSProfile != "default" {
		t.Errorf("global aws_profile = %q, want %q", cfg.AWSProfile, "default")
	}
	if cfg.AWSRegion != "ap-southeast-2" {
		t.Errorf("global aws_region = %q, want %q", cfg.AWSRegion, "ap-southeast-2")
	}
	if len(cfg.Databases) != 2 {
		t.Fatalf("got %d databases, want 2", len(cfg.Databases))
	}
	if cfg.Databases[0].Name != "Prod Main" {
		t.Errorf("db[0].name = %q, want %q", cfg.Databases[0].Name, "Prod Main")
	}
	if cfg.Databases[1].AWSProfile != "other" {
		t.Errorf("db[1].aws_profile = %q, want %q", cfg.Databases[1].AWSProfile, "other")
	}
}

func TestLoadConfig_DefaultMerging(t *testing.T) {
	configPath := writeTestConfig(t, t.TempDir(), `
aws_profile = "global-profile"
aws_region = "global-region"

[[database]]
name = "DB1"
database = "db1"
port = 7700

[[database]]
name = "DB2"
database = "db2"
port = 7701
aws_profile = "override-profile"
aws_region = "override-region"
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	db1 := cfg.Databases[0]
	if db1.AWSProfile != "global-profile" {
		t.Errorf("db1.aws_profile = %q, want %q", db1.AWSProfile, "global-profile")
	}
	if db1.AWSRegion != "global-region" {
		t.Errorf("db1.aws_region = %q, want %q", db1.AWSRegion, "global-region")
	}

	db2 := cfg.Databases[1]
	if db2.AWSProfile != "override-profile" {
		t.Errorf("db2.aws_profile = %q, want %q", db2.AWSProfile, "override-profile")
	}
	if db2.AWSRegion != "override-region" {
		t.Errorf("db2.aws_region = %q, want %q", db2.AWSRegion, "override-region")
	}
}

func TestLoadConfig_NoDatabases(t *testing.T) {
	configPath := writeTestConfig(t, t.TempDir(), `aws_profile = "default"`)

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("expected error for empty databases, got nil")
	}
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing name",
			content: `
[[database]]
database = "db1"
port = 7700
`,
		},
		{
			name: "missing database",
			content: `
[[database]]
name = "DB1"
port = 7700
`,
		},
		{
			name: "missing port",
			content: `
[[database]]
name = "DB1"
database = "db1"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := writeTestConfig(t, t.TempDir(), tt.content)

			_, err := LoadConfig(configPath)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestLoadConfig_AllFieldsParsed(t *testing.T) {
	configPath := writeTestConfig(t, t.TempDir(), `
verbose = true
license = "ABC-123"
aws_profile = "default"
aws_region = "ap-southeast-2"
security_group = "sg-global"
subnet = "subnet-global"
ttl = 4
forever = true

[[database]]
name = "Full Config"
database = "full-db"
port = 7700
aws_profile = "custom"
aws_region = "eu-west-1"
security_group = "sg-custom"
subnet = "subnet-custom"
ttl = 8
forever = false
elasticache = true
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Verbose {
		t.Error("verbose should be true")
	}
	if cfg.License != "ABC-123" {
		t.Errorf("license = %q, want %q", cfg.License, "ABC-123")
	}

	db := cfg.Databases[0]
	if db.AWSProfile != "custom" {
		t.Errorf("aws_profile = %q, want %q", db.AWSProfile, "custom")
	}
	if db.SecurityGroup != "sg-custom" {
		t.Errorf("security_group = %q, want %q", db.SecurityGroup, "sg-custom")
	}
	if db.Subnet != "subnet-custom" {
		t.Errorf("subnet = %q, want %q", db.Subnet, "subnet-custom")
	}
	if db.TTL != 8 {
		t.Errorf("ttl = %d, want %d", db.TTL, 8)
	}
	if db.Forever == nil || *db.Forever != false {
		t.Error("forever should be explicitly false")
	}
	if !db.Elasticache {
		t.Error("elasticache should be true")
	}
}

func TestLoadConfig_NewFieldDefaultMerging(t *testing.T) {
	configPath := writeTestConfig(t, t.TempDir(), `
security_group = "sg-global"
subnet = "subnet-global"
ttl = 4
forever = true

[[database]]
name = "Inherits"
database = "db1"
port = 7700

[[database]]
name = "Overrides"
database = "db2"
port = 7701
security_group = "sg-override"
subnet = "subnet-override"
ttl = 8
forever = false
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DB1 should inherit all globals
	db1 := cfg.Databases[0]
	if db1.SecurityGroup != "sg-global" {
		t.Errorf("db1 security_group = %q, want %q", db1.SecurityGroup, "sg-global")
	}
	if db1.Subnet != "subnet-global" {
		t.Errorf("db1 subnet = %q, want %q", db1.Subnet, "subnet-global")
	}
	if db1.TTL != 4 {
		t.Errorf("db1 ttl = %d, want %d", db1.TTL, 4)
	}
	if db1.Forever == nil || *db1.Forever != true {
		t.Error("db1 forever should inherit global true")
	}

	// DB2 should keep its overrides
	db2 := cfg.Databases[1]
	if db2.SecurityGroup != "sg-override" {
		t.Errorf("db2 security_group = %q, want %q", db2.SecurityGroup, "sg-override")
	}
	if db2.Subnet != "subnet-override" {
		t.Errorf("db2 subnet = %q, want %q", db2.Subnet, "subnet-override")
	}
	if db2.TTL != 8 {
		t.Errorf("db2 ttl = %d, want %d", db2.TTL, 8)
	}
	if db2.Forever == nil || *db2.Forever != false {
		t.Error("db2 forever should be explicitly false")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.toml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
