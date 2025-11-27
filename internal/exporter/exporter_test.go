package exporter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/miyanaga/backlog-exporter/internal/backlog"
	"github.com/miyanaga/backlog-exporter/internal/config"
)

// testOutput はテスト用の出力バッファ
type testOutput struct {
	buffer strings.Builder
}

func (t *testOutput) Printf(format string, args ...interface{}) {
	// テスト中は出力を抑制
}

func createTestData() (*backlog.Project, []*backlog.Status, []*backlog.Issue) {
	project := &backlog.Project{
		ID:         1,
		ProjectKey: "MYPROJ",
		Name:       "マイプロジェクト",
	}

	statuses := []*backlog.Status{
		{ID: 1, Name: "未対応"},
		{ID: 2, Name: "処理中"},
		{ID: 3, Name: "処理済み"},
		{ID: 4, Name: "完了"},
	}

	dueDate := "2024-12-01"
	parentID := 100
	issues := []*backlog.Issue{
		{
			ID:        100,
			IssueKey:  "MYPROJ-100",
			Summary:   "親課題1",
			Status:    &backlog.Status{ID: 2, Name: "処理中"},
			Priority:  &backlog.Priority{ID: 2, Name: "高"},
			Assignee:  &backlog.User{ID: 1, Name: "山田"},
			DueDate:   &dueDate,
			Created:   time.Date(2024, 11, 1, 10, 0, 0, 0, time.UTC),
			Updated:   time.Date(2024, 11, 25, 15, 30, 0, 0, time.UTC),
		},
		{
			ID:            101,
			IssueKey:      "MYPROJ-101",
			Summary:       "子課題1-1",
			Status:        &backlog.Status{ID: 3, Name: "処理済み"},
			Priority:      &backlog.Priority{ID: 2, Name: "高"},
			Assignee:      &backlog.User{ID: 1, Name: "山田"},
			ParentIssueID: &parentID,
			Created:       time.Date(2024, 11, 2, 10, 0, 0, 0, time.UTC),
			Updated:       time.Date(2024, 11, 14, 15, 30, 0, 0, time.UTC),
		},
		{
			ID:        200,
			IssueKey:  "MYPROJ-200",
			Summary:   "単独タスク",
			Status:    &backlog.Status{ID: 1, Name: "未対応"},
			Priority:  &backlog.Priority{ID: 3, Name: "中"},
			Created:   time.Date(2024, 11, 15, 10, 0, 0, 0, time.UTC),
			Updated:   time.Date(2024, 11, 20, 15, 30, 0, 0, time.UTC),
		},
	}

	return project, statuses, issues
}

func TestExporter_Run(t *testing.T) {
	project, statuses, issues := createTestData()

	mockClient := &backlog.MockClient{
		GetProjectFunc: func(ctx context.Context, projectIDOrKey string) (*backlog.Project, error) {
			return project, nil
		},
		GetStatusesFunc: func(ctx context.Context, projectIDOrKey string) ([]*backlog.Status, error) {
			return statuses, nil
		},
		GetIssuesFunc: func(ctx context.Context, projectID int, statusIDs []int, assigneeID *int, progressFn func(fetched, total int)) ([]*backlog.Issue, error) {
			// 完了以外の状態のみを要求しているか確認
			for _, id := range statusIDs {
				if id == 4 {
					t.Error("status ID 4 (完了) should not be requested")
				}
			}
			return issues, nil
		},
	}

	// テンポラリディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "backlog-exporter-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		APIKey:  "test-api-key",
		Space:   "mycompany",
		Domain:  "backlog.com",
		Project: "MYPROJ",
		Output:  tmpDir,
		Format:  config.FormatTXT,
	}

	exp := NewExporterWithOutput(mockClient, cfg, &testOutput{})
	ctx := context.Background()

	outputPath, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ファイルが作成されたか確認
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("output file was not created: %s", outputPath)
	}

	// ファイル名の形式を確認
	filename := filepath.Base(outputPath)
	if !strings.HasPrefix(filename, "MYPROJ_tasks_") {
		t.Errorf("unexpected filename: %s", filename)
	}
	if !strings.HasSuffix(filename, ".txt") {
		t.Errorf("unexpected file extension: %s", filename)
	}

	// ファイル内容を確認
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "MYPROJ") {
		t.Error("output should contain project key")
	}
	if !strings.Contains(contentStr, "マイプロジェクト") {
		t.Error("output should contain project name")
	}
	if !strings.Contains(contentStr, "親課題1") {
		t.Error("output should contain parent issue")
	}
	if !strings.Contains(contentStr, "子課題1-1") {
		t.Error("output should contain child issue")
	}
}

func TestExporter_BuildHierarchy(t *testing.T) {
	_, _, issues := createTestData()

	cfg := &config.Config{
		Format: config.FormatTXT,
	}
	exp := NewExporter(&backlog.MockClient{}, cfg)

	hierarchical, summary := exp.buildHierarchy(issues)

	// 親課題（またはスタンドアロン課題）は2つ
	if len(hierarchical) != 2 {
		t.Errorf("expected 2 root issues, got %d", len(hierarchical))
	}

	// サマリーの確認
	if summary.Total != 3 {
		t.Errorf("expected total 3, got %d", summary.Total)
	}
	if summary.ParentIssues != 2 {
		t.Errorf("expected 2 parent issues, got %d", summary.ParentIssues)
	}
	if summary.ChildIssues != 1 {
		t.Errorf("expected 1 child issue, got %d", summary.ChildIssues)
	}

	// 最初の親課題に子課題があるか確認
	found := false
	for _, hi := range hierarchical {
		if hi.Issue.IssueKey == "MYPROJ-100" {
			found = true
			if len(hi.Children) != 1 {
				t.Errorf("expected 1 child, got %d", len(hi.Children))
			}
			if hi.Children[0].Issue.IssueKey != "MYPROJ-101" {
				t.Errorf("expected child MYPROJ-101, got %s", hi.Children[0].Issue.IssueKey)
			}
		}
	}
	if !found {
		t.Error("MYPROJ-100 not found in root issues")
	}
}

func TestExporter_OutputFormats(t *testing.T) {
	project, statuses, issues := createTestData()

	mockClient := &backlog.MockClient{
		GetProjectFunc: func(ctx context.Context, projectIDOrKey string) (*backlog.Project, error) {
			return project, nil
		},
		GetStatusesFunc: func(ctx context.Context, projectIDOrKey string) ([]*backlog.Status, error) {
			return statuses, nil
		},
		GetIssuesFunc: func(ctx context.Context, projectID int, statusIDs []int, assigneeID *int, progressFn func(fetched, total int)) ([]*backlog.Issue, error) {
			return issues, nil
		},
	}

	formats := []struct {
		format    config.OutputFormat
		extension string
		contains  []string
	}{
		{
			format:    config.FormatTXT,
			extension: ".txt",
			contains:  []string{"================", "MYPROJ-100", "└─"},
		},
		{
			format:    config.FormatMarkdown,
			extension: ".md",
			contains:  []string{"# MYPROJ", "## [MYPROJ-100]", "| 項目 | 内容 |", "### 子課題"},
		},
		{
			format:    config.FormatJSON,
			extension: ".json",
			contains:  []string{`"issueKey"`, `"MYPROJ-100"`, `"children"`, `"summary"`},
		},
	}

	for _, tc := range formats {
		t.Run(string(tc.format), func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "backlog-exporter-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			cfg := &config.Config{
				APIKey:  "test-api-key",
				Space:   "mycompany",
				Domain:  "backlog.com",
				Project: "MYPROJ",
				Output:  tmpDir,
				Format:  tc.format,
			}

			exp := NewExporterWithOutput(mockClient, cfg, &testOutput{})
			ctx := context.Background()

			outputPath, err := exp.Run(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// 拡張子の確認
			if !strings.HasSuffix(outputPath, tc.extension) {
				t.Errorf("expected extension %s, got %s", tc.extension, filepath.Ext(outputPath))
			}

			// 内容の確認
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			contentStr := string(content)
			for _, expected := range tc.contains {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("output should contain %q", expected)
				}
			}
		})
	}
}
