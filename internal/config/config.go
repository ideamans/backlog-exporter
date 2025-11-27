package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

// OutputFormat は出力フォーマットを表す
type OutputFormat string

const (
	FormatTXT      OutputFormat = "txt"
	FormatJSON     OutputFormat = "json"
	FormatMarkdown OutputFormat = "markdown"
)

// Config はCLIの設定を表す
type Config struct {
	APIKey   string
	Space    string
	Domain   string
	Project  string
	Output   string
	Format   OutputFormat
	Assignee *int
}

// Validate は設定を検証する
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("API key is required. Set --api-key or BACKLOG_API_KEY")
	}
	if c.Space == "" {
		return errors.New("space is required. Use --space or -s")
	}
	if c.Project == "" {
		return errors.New("project is required. Use --project or -p")
	}
	if c.Domain == "" {
		c.Domain = "backlog.com"
	}
	if c.Output == "" {
		c.Output = "./"
	}
	if c.Format == "" {
		c.Format = FormatTXT
	}

	// フォーマットの検証
	switch c.Format {
	case FormatTXT, FormatJSON, FormatMarkdown:
		// OK
	default:
		return fmt.Errorf("invalid format: %s. Use txt, json, or markdown", c.Format)
	}

	return nil
}

// IsProjectID はプロジェクト指定が数値（ID）かどうかを判定する
func (c *Config) IsProjectID() bool {
	for _, r := range c.Project {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(c.Project) > 0
}

// GetProjectID はプロジェクトIDを取得する（数値の場合のみ）
func (c *Config) GetProjectID() (int, error) {
	if !c.IsProjectID() {
		return 0, errors.New("project is not a numeric ID")
	}
	return strconv.Atoi(c.Project)
}

// LoadFromEnv は環境変数から設定を読み込む
func LoadFromEnv() *Config {
	cfg := &Config{}

	cfg.APIKey = os.Getenv("BACKLOG_API_KEY")
	cfg.Space = os.Getenv("BACKLOG_SPACE")
	cfg.Domain = os.Getenv("BACKLOG_DOMAIN")

	return cfg
}

// Merge は他の設定をマージする（空でない値で上書き）
func (c *Config) Merge(other *Config) {
	if other.APIKey != "" {
		c.APIKey = other.APIKey
	}
	if other.Space != "" {
		c.Space = other.Space
	}
	if other.Domain != "" {
		c.Domain = other.Domain
	}
	if other.Project != "" {
		c.Project = other.Project
	}
	if other.Output != "" {
		c.Output = other.Output
	}
	if other.Format != "" {
		c.Format = other.Format
	}
	if other.Assignee != nil {
		c.Assignee = other.Assignee
	}
}
