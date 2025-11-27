package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/miyanaga/backlog-exporter/internal/backlog"
	"github.com/miyanaga/backlog-exporter/internal/config"
	"github.com/miyanaga/backlog-exporter/internal/exporter"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Exit codes
const (
	ExitSuccess           = 0
	ExitAPIKeyRequired    = 1
	ExitAuthError         = 2
	ExitProjectNotFound   = 3
	ExitNetworkError      = 4
	ExitOutputDirError    = 5
	ExitRateLimitExceeded = 6
	ExitInvalidArgs       = 7
)

func main() {
	os.Exit(run())
}

func run() int {
	// フラグの定義
	var (
		apiKey      string
		space       string
		domain      string
		project     string
		output      string
		format      string
		assignee    int
		showHelp    bool
		showVersion bool
	)

	flag.StringVar(&apiKey, "api-key", "", "Backlog API key")
	flag.StringVar(&apiKey, "k", "", "Backlog API key (shorthand)")
	flag.StringVar(&space, "space", "", "Backlog space ID (e.g., mycompany)")
	flag.StringVar(&space, "s", "", "Backlog space ID (shorthand)")
	flag.StringVar(&domain, "domain", "backlog.com", "Backlog domain (backlog.com, backlog.jp, backlogtool.com)")
	flag.StringVar(&domain, "d", "backlog.com", "Backlog domain (shorthand)")
	flag.StringVar(&project, "project", "", "Project ID or project key")
	flag.StringVar(&project, "p", "", "Project ID or project key (shorthand)")
	flag.StringVar(&output, "output", "./", "Output directory")
	flag.StringVar(&output, "o", "./", "Output directory (shorthand)")
	flag.StringVar(&format, "format", "txt", "Output format (txt, json, markdown)")
	flag.StringVar(&format, "f", "txt", "Output format (shorthand)")
	flag.IntVar(&assignee, "assignee", 0, "Filter by assignee user ID")
	flag.IntVar(&assignee, "a", 0, "Filter by assignee user ID (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: backlog-tasks [options]\n\n")
		fmt.Fprintf(os.Stderr, "A CLI tool to export incomplete tasks from Backlog.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -k, --api-key    Backlog API key (or set BACKLOG_API_KEY)\n")
		fmt.Fprintf(os.Stderr, "  -s, --space      Backlog space ID (required)\n")
		fmt.Fprintf(os.Stderr, "  -d, --domain     Backlog domain (default: backlog.com)\n")
		fmt.Fprintf(os.Stderr, "  -p, --project    Project ID or key (required)\n")
		fmt.Fprintf(os.Stderr, "  -o, --output     Output directory (default: ./)\n")
		fmt.Fprintf(os.Stderr, "  -f, --format     Output format: txt, json, markdown (default: txt)\n")
		fmt.Fprintf(os.Stderr, "  -a, --assignee   Filter by assignee user ID\n")
		fmt.Fprintf(os.Stderr, "  -h, --help       Show this help message\n")
		fmt.Fprintf(os.Stderr, "  -v, --version    Show version\n\n")
		fmt.Fprintf(os.Stderr, "Environment variables:\n")
		fmt.Fprintf(os.Stderr, "  BACKLOG_API_KEY  Backlog API key\n")
		fmt.Fprintf(os.Stderr, "  BACKLOG_SPACE    Backlog space ID\n")
		fmt.Fprintf(os.Stderr, "  BACKLOG_DOMAIN   Backlog domain\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  # Export to Markdown\n")
		fmt.Fprintf(os.Stderr, "  backlog-tasks -s mycompany -p MYPROJ -f markdown\n\n")
		fmt.Fprintf(os.Stderr, "  # Export with API key\n")
		fmt.Fprintf(os.Stderr, "  backlog-tasks -k YOUR_API_KEY -s mycompany -p MYPROJ\n")
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		return ExitSuccess
	}

	if showVersion {
		fmt.Printf("backlog-tasks %s (commit: %s, built: %s)\n", version, commit, date)
		return ExitSuccess
	}

	// 環境変数から設定を読み込み
	cfg := config.LoadFromEnv()

	// コマンドライン引数で上書き
	cmdCfg := &config.Config{
		APIKey:  apiKey,
		Space:   space,
		Domain:  domain,
		Project: project,
		Output:  output,
		Format:  config.OutputFormat(format),
	}
	if assignee > 0 {
		cmdCfg.Assignee = &assignee
	}

	cfg.Merge(cmdCfg)

	// バリデーション
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return ExitInvalidArgs
	}

	// 出力ディレクトリの確認
	if _, err := os.Stat(cfg.Output); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Cannot write to directory '%s'\n", cfg.Output)
		return ExitOutputDirError
	}

	// APIクライアントの作成
	client := backlog.NewClient(cfg.Space, cfg.Domain, cfg.APIKey)

	// エクスポーターの作成と実行
	exp := exporter.NewExporter(client, cfg)
	ctx := context.Background()

	if _, err := exp.Run(ctx); err != nil {
		// エラーの種類に応じて終了コードを設定
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return classifyError(err)
	}

	return ExitSuccess
}

func classifyError(err error) int {
	// エラーメッセージに基づいて終了コードを分類
	errStr := err.Error()

	switch {
	case contains(errStr, "API key"):
		return ExitAPIKeyRequired
	case contains(errStr, "Authentication failed"), contains(errStr, "401"):
		return ExitAuthError
	case contains(errStr, "not found"), contains(errStr, "404"):
		return ExitProjectNotFound
	case contains(errStr, "connect"), contains(errStr, "network"):
		return ExitNetworkError
	case contains(errStr, "rate limit"), contains(errStr, "429"):
		return ExitRateLimitExceeded
	default:
		return ExitInvalidArgs
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
