package exporter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/miyanaga/backlog-exporter/internal/backlog"
	"github.com/miyanaga/backlog-exporter/internal/config"
)

// Integration tests using HTTP mock server and temp directory

func setupMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/projects/TESTPROJ"):
			// プロジェクト情報を返す
			project := backlog.Project{
				ID:         100,
				ProjectKey: "TESTPROJ",
				Name:       "テストプロジェクト",
			}
			json.NewEncoder(w).Encode(project)

		case strings.HasSuffix(r.URL.Path, "/projects/TESTPROJ/statuses"):
			// ステータス一覧を返す
			statuses := []*backlog.Status{
				{ID: 1, Name: "未対応"},
				{ID: 2, Name: "処理中"},
				{ID: 3, Name: "処理済み"},
				{ID: 4, Name: "完了"},
			}
			json.NewEncoder(w).Encode(statuses)

		case strings.HasSuffix(r.URL.Path, "/issues"):
			// 課題一覧を返す
			dueDate1 := "2024-12-15"
			dueDate2 := "2024-12-20"
			parentID := 1

			issues := []*backlog.Issue{
				{
					ID:        1,
					IssueKey:  "TESTPROJ-1",
					Summary:   "親課題: システム設計",
					Status:    &backlog.Status{ID: 2, Name: "処理中"},
					Priority:  &backlog.Priority{ID: 2, Name: "高"},
					Assignee:  &backlog.User{ID: 1, Name: "田中太郎"},
					DueDate:   &dueDate1,
					Created:   time.Date(2024, 11, 1, 9, 0, 0, 0, time.UTC),
					Updated:   time.Date(2024, 11, 26, 17, 30, 0, 0, time.UTC),
				},
				{
					ID:            2,
					IssueKey:      "TESTPROJ-2",
					Summary:       "子課題: DB設計",
					Status:        &backlog.Status{ID: 3, Name: "処理済み"},
					Priority:      &backlog.Priority{ID: 2, Name: "高"},
					Assignee:      &backlog.User{ID: 1, Name: "田中太郎"},
					ParentIssueID: &parentID,
					Created:       time.Date(2024, 11, 5, 10, 0, 0, 0, time.UTC),
					Updated:       time.Date(2024, 11, 20, 15, 0, 0, 0, time.UTC),
				},
				{
					ID:            3,
					IssueKey:      "TESTPROJ-3",
					Summary:       "子課題: API設計",
					Status:        &backlog.Status{ID: 2, Name: "処理中"},
					Priority:      &backlog.Priority{ID: 3, Name: "中"},
					Assignee:      &backlog.User{ID: 2, Name: "鈴木花子"},
					ParentIssueID: &parentID,
					DueDate:       &dueDate2,
					Created:       time.Date(2024, 11, 5, 10, 30, 0, 0, time.UTC),
					Updated:       time.Date(2024, 11, 25, 14, 0, 0, 0, time.UTC),
				},
				{
					ID:       4,
					IssueKey: "TESTPROJ-4",
					Summary:  "ドキュメント作成",
					Status:   &backlog.Status{ID: 1, Name: "未対応"},
					Priority: &backlog.Priority{ID: 4, Name: "低"},
					Created:  time.Date(2024, 11, 10, 11, 0, 0, 0, time.UTC),
					Updated:  time.Date(2024, 11, 10, 11, 0, 0, 0, time.UTC),
				},
			}
			json.NewEncoder(w).Encode(issues)

		default:
			http.NotFound(w, r)
		}
	}))
}

func TestIntegration_ExportToTempDirectory(t *testing.T) {
	// モックサーバーをセットアップ
	server := setupMockServer(t)
	defer server.Close()

	// テンポラリディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "backlog-exporter-integration")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// APIクライアントを作成（モックサーバーを使用）
	client := backlog.NewClientWithHTTPClient(
		server.URL+"/api/v2",
		"test-api-key",
		server.Client(),
	)

	// テストケース
	testCases := []struct {
		name      string
		format    config.OutputFormat
		extension string
		validate  func(t *testing.T, content string)
	}{
		{
			name:      "TXT format",
			format:    config.FormatTXT,
			extension: ".txt",
			validate: func(t *testing.T, content string) {
				// ヘッダーの検証
				if !strings.Contains(content, "TESTPROJ - テストプロジェクト") {
					t.Error("should contain project info")
				}
				if !strings.Contains(content, "未完了タスク数: 4件") {
					t.Error("should contain task count")
				}

				// 親課題の検証
				if !strings.Contains(content, "[TESTPROJ-1] 親課題: システム設計") {
					t.Error("should contain parent issue")
				}
				if !strings.Contains(content, "担当者: 田中太郎") {
					t.Error("should contain assignee")
				}

				// 子課題の検証（ツリー形式）
				if !strings.Contains(content, "├─ [TESTPROJ-2]") || !strings.Contains(content, "└─ [TESTPROJ-3]") {
					t.Error("should contain child issues with tree markers")
				}

				// 単独課題の検証
				if !strings.Contains(content, "[TESTPROJ-4] ドキュメント作成") {
					t.Error("should contain standalone issue")
				}
				if !strings.Contains(content, "担当者: (未割当)") {
					t.Error("should show unassigned for tasks without assignee")
				}
			},
		},
		{
			name:      "Markdown format",
			format:    config.FormatMarkdown,
			extension: ".md",
			validate: func(t *testing.T, content string) {
				// タイトルの検証
				if !strings.Contains(content, "# TESTPROJ - テストプロジェクト 未完了タスク一覧") {
					t.Error("should contain markdown title")
				}

				// テーブルの検証
				if !strings.Contains(content, "| 項目 | 内容 |") {
					t.Error("should contain table header")
				}
				if !strings.Contains(content, "| 状態 | 処理中 |") {
					t.Error("should contain status in table")
				}

				// 子課題セクションの検証
				if !strings.Contains(content, "### 子課題") {
					t.Error("should contain child issues section")
				}
				if !strings.Contains(content, "#### [TESTPROJ-2]") {
					t.Error("should contain child issue as h4")
				}
			},
		},
		{
			name:      "JSON format",
			format:    config.FormatJSON,
			extension: ".json",
			validate: func(t *testing.T, content string) {
				// JSONとしてパース可能か検証
				var data jsonExportData
				if err := json.Unmarshal([]byte(content), &data); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}

				// プロジェクト情報の検証
				if data.Project.Key != "TESTPROJ" {
					t.Errorf("expected project key TESTPROJ, got %s", data.Project.Key)
				}

				// サマリーの検証
				if data.Summary.Total != 4 {
					t.Errorf("expected total 4, got %d", data.Summary.Total)
				}
				if data.Summary.ParentIssues != 2 {
					t.Errorf("expected 2 parent issues, got %d", data.Summary.ParentIssues)
				}
				if data.Summary.ChildIssues != 2 {
					t.Errorf("expected 2 child issues, got %d", data.Summary.ChildIssues)
				}

				// 階層構造の検証
				var parentIssue *jsonIssue
				for i := range data.Issues {
					if data.Issues[i].IssueKey == "TESTPROJ-1" {
						parentIssue = &data.Issues[i]
						break
					}
				}
				if parentIssue == nil {
					t.Fatal("parent issue TESTPROJ-1 not found")
				}
				if len(parentIssue.Children) != 2 {
					t.Errorf("expected 2 children, got %d", len(parentIssue.Children))
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				APIKey:  "test-api-key",
				Space:   "test",
				Domain:  "backlog.com",
				Project: "TESTPROJ",
				Output:  tmpDir,
				Format:  tc.format,
			}

			exp := NewExporterWithOutput(client, cfg, &testOutput{})
			ctx := context.Background()

			outputPath, err := exp.Run(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// ファイルが作成されたか確認
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Fatalf("output file was not created: %s", outputPath)
			}

			// 拡張子の確認
			if !strings.HasSuffix(outputPath, tc.extension) {
				t.Errorf("expected extension %s, got %s", tc.extension, filepath.Ext(outputPath))
			}

			// ファイル内容を読み込んで検証
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			tc.validate(t, string(content))
		})
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	// エラーレスポンスを返すモックサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		apiErr := backlog.APIError{
			Errors: []struct {
				Message  string `json:"message"`
				Code     int    `json:"code"`
				MoreInfo string `json:"moreInfo"`
			}{
				{Message: "Project not found", Code: 6},
			},
		}
		json.NewEncoder(w).Encode(apiErr)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "backlog-exporter-integration-error")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	client := backlog.NewClientWithHTTPClient(
		server.URL+"/api/v2",
		"test-api-key",
		server.Client(),
	)

	cfg := &config.Config{
		APIKey:  "test-api-key",
		Space:   "test",
		Domain:  "backlog.com",
		Project: "NOTFOUND",
		Output:  tmpDir,
		Format:  config.FormatTXT,
	}

	exp := NewExporterWithOutput(client, cfg, &testOutput{})
	ctx := context.Background()

	_, err = exp.Run(ctx)
	if err == nil {
		t.Error("expected error for non-existent project")
	}
}

func TestIntegration_EmptyIssues(t *testing.T) {
	// 空の課題一覧を返すモックサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/projects/EMPTYPROJ"):
			project := backlog.Project{
				ID:         200,
				ProjectKey: "EMPTYPROJ",
				Name:       "空のプロジェクト",
			}
			json.NewEncoder(w).Encode(project)

		case strings.HasSuffix(r.URL.Path, "/projects/EMPTYPROJ/statuses"):
			statuses := []*backlog.Status{
				{ID: 1, Name: "未対応"},
				{ID: 4, Name: "完了"},
			}
			json.NewEncoder(w).Encode(statuses)

		case strings.HasSuffix(r.URL.Path, "/issues"):
			// 空の課題一覧
			json.NewEncoder(w).Encode([]*backlog.Issue{})

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "backlog-exporter-integration-empty")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	client := backlog.NewClientWithHTTPClient(
		server.URL+"/api/v2",
		"test-api-key",
		server.Client(),
	)

	cfg := &config.Config{
		APIKey:  "test-api-key",
		Space:   "test",
		Domain:  "backlog.com",
		Project: "EMPTYPROJ",
		Output:  tmpDir,
		Format:  config.FormatJSON,
	}

	exp := NewExporterWithOutput(client, cfg, &testOutput{})
	ctx := context.Background()

	outputPath, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ファイル内容を検証
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var data jsonExportData
	if err := json.Unmarshal(content, &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if data.Summary.Total != 0 {
		t.Errorf("expected total 0, got %d", data.Summary.Total)
	}
	if len(data.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(data.Issues))
	}
}
