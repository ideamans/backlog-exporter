package config

import (
	"os"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				APIKey:  "test-key",
				Space:   "mycompany",
				Project: "MYPROJ",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &Config{
				Space:   "mycompany",
				Project: "MYPROJ",
			},
			wantErr: true,
		},
		{
			name: "missing space",
			config: &Config{
				APIKey:  "test-key",
				Project: "MYPROJ",
			},
			wantErr: true,
		},
		{
			name: "missing project",
			config: &Config{
				APIKey: "test-key",
				Space:  "mycompany",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: &Config{
				APIKey:  "test-key",
				Space:   "mycompany",
				Project: "MYPROJ",
				Format:  "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid with all formats",
			config: &Config{
				APIKey:  "test-key",
				Space:   "mycompany",
				Project: "MYPROJ",
				Format:  FormatMarkdown,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestConfig_Validate_Defaults(t *testing.T) {
	cfg := &Config{
		APIKey:  "test-key",
		Space:   "mycompany",
		Project: "MYPROJ",
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Domain != "backlog.com" {
		t.Errorf("expected default domain backlog.com, got %s", cfg.Domain)
	}
	if cfg.Output != "./" {
		t.Errorf("expected default output ./, got %s", cfg.Output)
	}
	if cfg.Format != FormatTXT {
		t.Errorf("expected default format txt, got %s", cfg.Format)
	}
}

func TestConfig_IsProjectID(t *testing.T) {
	tests := []struct {
		project  string
		expected bool
	}{
		{"12345", true},
		{"0", true},
		{"MYPROJ", false},
		{"MYPROJ-123", false},
		{"proj123", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.project, func(t *testing.T) {
			cfg := &Config{Project: tc.project}
			if got := cfg.IsProjectID(); got != tc.expected {
				t.Errorf("IsProjectID() = %v, expected %v", got, tc.expected)
			}
		})
	}
}

func TestConfig_GetProjectID(t *testing.T) {
	cfg := &Config{Project: "12345"}
	id, err := cfg.GetProjectID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 12345 {
		t.Errorf("expected 12345, got %d", id)
	}

	cfg2 := &Config{Project: "MYPROJ"}
	_, err = cfg2.GetProjectID()
	if err == nil {
		t.Error("expected error for non-numeric project")
	}
}

func TestLoadFromEnv(t *testing.T) {
	// 環境変数を設定
	os.Setenv("BACKLOG_API_KEY", "env-api-key")
	os.Setenv("BACKLOG_SPACE", "env-space")
	os.Setenv("BACKLOG_DOMAIN", "backlog.jp")
	defer func() {
		os.Unsetenv("BACKLOG_API_KEY")
		os.Unsetenv("BACKLOG_SPACE")
		os.Unsetenv("BACKLOG_DOMAIN")
	}()

	cfg := LoadFromEnv()

	if cfg.APIKey != "env-api-key" {
		t.Errorf("expected APIKey env-api-key, got %s", cfg.APIKey)
	}
	if cfg.Space != "env-space" {
		t.Errorf("expected Space env-space, got %s", cfg.Space)
	}
	if cfg.Domain != "backlog.jp" {
		t.Errorf("expected Domain backlog.jp, got %s", cfg.Domain)
	}
}

func TestConfig_Merge(t *testing.T) {
	assignee := 123
	base := &Config{
		APIKey: "base-key",
		Space:  "base-space",
		Domain: "base-domain",
	}

	other := &Config{
		APIKey:   "other-key",
		Project:  "other-project",
		Output:   "./output",
		Format:   FormatJSON,
		Assignee: &assignee,
	}

	base.Merge(other)

	if base.APIKey != "other-key" {
		t.Errorf("APIKey should be merged, got %s", base.APIKey)
	}
	if base.Space != "base-space" {
		t.Errorf("Space should not be overwritten by empty value, got %s", base.Space)
	}
	if base.Domain != "base-domain" {
		t.Errorf("Domain should not be overwritten by empty value, got %s", base.Domain)
	}
	if base.Project != "other-project" {
		t.Errorf("Project should be merged, got %s", base.Project)
	}
	if base.Output != "./output" {
		t.Errorf("Output should be merged, got %s", base.Output)
	}
	if base.Format != FormatJSON {
		t.Errorf("Format should be merged, got %s", base.Format)
	}
	if base.Assignee == nil || *base.Assignee != 123 {
		t.Errorf("Assignee should be merged")
	}
}
