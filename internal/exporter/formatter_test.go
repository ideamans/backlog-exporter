package exporter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/miyanaga/backlog-exporter/internal/backlog"
	"github.com/miyanaga/backlog-exporter/internal/config"
)

func createTestExportData() *backlog.ExportData {
	dueDate := "2024-12-01"
	return &backlog.ExportData{
		Project: &backlog.Project{
			ID:         1,
			ProjectKey: "MYPROJ",
			Name:       "マイプロジェクト",
		},
		ExportedAt: time.Date(2024, 11, 27, 14, 30, 52, 0, time.Local),
		Summary: backlog.ExportSummary{
			Total:        3,
			ParentIssues: 2,
			ChildIssues:  1,
		},
		Issues: []*backlog.HierarchicalIssue{
			{
				Issue: &backlog.Issue{
					ID:       100,
					IssueKey: "MYPROJ-100",
					Summary:  "親課題",
					Status:   &backlog.Status{ID: 2, Name: "処理中"},
					Priority: &backlog.Priority{ID: 2, Name: "高"},
					Assignee: &backlog.User{ID: 1, Name: "山田"},
					DueDate:  &dueDate,
					Created:  time.Date(2024, 11, 1, 10, 0, 0, 0, time.UTC),
					Updated:  time.Date(2024, 11, 25, 15, 30, 0, 0, time.UTC),
				},
				Children: []*backlog.HierarchicalIssue{
					{
						Issue: &backlog.Issue{
							ID:       101,
							IssueKey: "MYPROJ-101",
							Summary:  "子課題",
							Status:   &backlog.Status{ID: 3, Name: "処理済み"},
							Priority: &backlog.Priority{ID: 2, Name: "高"},
							Assignee: &backlog.User{ID: 2, Name: "鈴木"},
							Created:  time.Date(2024, 11, 2, 10, 0, 0, 0, time.UTC),
							Updated:  time.Date(2024, 11, 14, 15, 30, 0, 0, time.UTC),
						},
						Children: []*backlog.HierarchicalIssue{},
					},
				},
			},
			{
				Issue: &backlog.Issue{
					ID:       200,
					IssueKey: "MYPROJ-200",
					Summary:  "単独タスク",
					Status:   &backlog.Status{ID: 1, Name: "未対応"},
					Priority: &backlog.Priority{ID: 3, Name: "中"},
					Created:  time.Date(2024, 11, 15, 10, 0, 0, 0, time.UTC),
					Updated:  time.Date(2024, 11, 20, 15, 30, 0, 0, time.UTC),
				},
				Children: []*backlog.HierarchicalIssue{},
			},
		},
	}
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		format   config.OutputFormat
		expected string
	}{
		{config.FormatTXT, "txt"},
		{config.FormatJSON, "json"},
		{config.FormatMarkdown, "md"},
	}

	for _, tc := range tests {
		t.Run(string(tc.format), func(t *testing.T) {
			f := NewFormatter(tc.format)
			if f.Extension() != tc.expected {
				t.Errorf("expected extension %s, got %s", tc.expected, f.Extension())
			}
		})
	}
}

func TestTXTFormatter_Format(t *testing.T) {
	data := createTestExportData()
	f := &TXTFormatter{}

	output, err := f.Format(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(output)

	// ヘッダーの確認
	if !strings.Contains(content, "================") {
		t.Error("should contain header separator")
	}
	if !strings.Contains(content, "MYPROJ - マイプロジェクト") {
		t.Error("should contain project info")
	}
	if !strings.Contains(content, "未完了タスク数: 3件") {
		t.Error("should contain task count")
	}

	// 課題の確認
	if !strings.Contains(content, "[MYPROJ-100] 親課題") {
		t.Error("should contain parent issue")
	}
	if !strings.Contains(content, "状態: 処理中") {
		t.Error("should contain status")
	}
	if !strings.Contains(content, "担当者: 山田") {
		t.Error("should contain assignee")
	}

	// 子課題の確認（ツリー表示）
	if !strings.Contains(content, "└─ [MYPROJ-101] 子課題") {
		t.Error("should contain child issue with tree marker")
	}

	// 単独タスクの確認
	if !strings.Contains(content, "[MYPROJ-200] 単独タスク") {
		t.Error("should contain standalone task")
	}
	if !strings.Contains(content, "担当者: (未割当)") {
		t.Error("should show unassigned for tasks without assignee")
	}
}

func TestMarkdownFormatter_Format(t *testing.T) {
	data := createTestExportData()
	f := &MarkdownFormatter{}

	output, err := f.Format(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(output)

	// ヘッダーの確認
	if !strings.Contains(content, "# MYPROJ - マイプロジェクト 未完了タスク一覧") {
		t.Error("should contain markdown title")
	}
	if !strings.Contains(content, "> 取得日時:") {
		t.Error("should contain export date")
	}

	// 課題の確認
	if !strings.Contains(content, "## [MYPROJ-100] 親課題") {
		t.Error("should contain parent issue as h2")
	}
	if !strings.Contains(content, "| 項目 | 内容 |") {
		t.Error("should contain table header")
	}
	if !strings.Contains(content, "| 状態 | 処理中 |") {
		t.Error("should contain status in table")
	}

	// 子課題の確認
	if !strings.Contains(content, "### 子課題") {
		t.Error("should contain child issues section")
	}
	if !strings.Contains(content, "#### [MYPROJ-101] 子課題") {
		t.Error("should contain child issue as h4")
	}

	// セパレーターの確認
	if !strings.Contains(content, "---") {
		t.Error("should contain markdown separators")
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	data := createTestExportData()
	f := &JSONFormatter{}

	output, err := f.Format(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// JSONとしてパース可能か確認
	var result jsonExportData
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// プロジェクト情報の確認
	if result.Project.Key != "MYPROJ" {
		t.Errorf("expected project key MYPROJ, got %s", result.Project.Key)
	}
	if result.Project.Name != "マイプロジェクト" {
		t.Errorf("expected project name マイプロジェクト, got %s", result.Project.Name)
	}

	// サマリーの確認
	if result.Summary.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Summary.Total)
	}
	if result.Summary.ParentIssues != 2 {
		t.Errorf("expected 2 parent issues, got %d", result.Summary.ParentIssues)
	}
	if result.Summary.ChildIssues != 1 {
		t.Errorf("expected 1 child issue, got %d", result.Summary.ChildIssues)
	}

	// 課題の確認
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 root issues, got %d", len(result.Issues))
	}

	// 親課題の確認
	parentFound := false
	for _, issue := range result.Issues {
		if issue.IssueKey == "MYPROJ-100" {
			parentFound = true
			if len(issue.Children) != 1 {
				t.Errorf("expected 1 child, got %d", len(issue.Children))
			}
			if issue.Children[0].IssueKey != "MYPROJ-101" {
				t.Errorf("expected child MYPROJ-101, got %s", issue.Children[0].IssueKey)
			}
			if issue.Status != "処理中" {
				t.Errorf("expected status 処理中, got %s", issue.Status)
			}
			if *issue.Assignee != "山田" {
				t.Errorf("expected assignee 山田, got %s", *issue.Assignee)
			}
		}
	}
	if !parentFound {
		t.Error("parent issue MYPROJ-100 not found")
	}

	// 担当者なしの場合の確認
	for _, issue := range result.Issues {
		if issue.IssueKey == "MYPROJ-200" {
			if issue.Assignee != nil {
				t.Error("expected nil assignee for MYPROJ-200")
			}
		}
	}
}

func TestJSONFormatter_Format_ValidJSON(t *testing.T) {
	data := createTestExportData()
	f := &JSONFormatter{}

	output, err := f.Format(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// JSONが整形されているか確認
	content := string(output)
	if !strings.Contains(content, "\n") {
		t.Error("JSON should be pretty-printed with newlines")
	}
	if !strings.Contains(content, "  ") {
		t.Error("JSON should be indented")
	}
}

func TestFormatter_NilValues(t *testing.T) {
	data := &backlog.ExportData{
		Project: &backlog.Project{
			ID:         1,
			ProjectKey: "TEST",
			Name:       "Test",
		},
		ExportedAt: time.Now(),
		Summary:    backlog.ExportSummary{Total: 1, ParentIssues: 1},
		Issues: []*backlog.HierarchicalIssue{
			{
				Issue: &backlog.Issue{
					ID:       1,
					IssueKey: "TEST-1",
					Summary:  "Test",
					// Status, Priority, Assignee, DueDate are all nil
					Created: time.Now(),
					Updated: time.Now(),
				},
				Children: []*backlog.HierarchicalIssue{},
			},
		},
	}

	formats := []config.OutputFormat{config.FormatTXT, config.FormatMarkdown, config.FormatJSON}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			f := NewFormatter(format)
			output, err := f.Format(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// nil値でもエラーにならないことを確認
			if len(output) == 0 {
				t.Error("output should not be empty")
			}
		})
	}
}
