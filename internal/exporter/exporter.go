package exporter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/miyanaga/backlog-exporter/internal/backlog"
	"github.com/miyanaga/backlog-exporter/internal/config"
)

// 完了状態のID（デフォルト）
const defaultCompletedStatusID = 4

// Exporter はBacklogタスクのエクスポートを行う
type Exporter struct {
	client    backlog.Client
	config    *config.Config
	formatter Formatter
	output    Output
}

// Output は出力先を抽象化するインターフェース
type Output interface {
	Printf(format string, args ...interface{})
}

// StdOutput は標準出力への出力
type StdOutput struct{}

func (s *StdOutput) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// NewExporter は新しいExporterを作成する
func NewExporter(client backlog.Client, cfg *config.Config) *Exporter {
	return &Exporter{
		client:    client,
		config:    cfg,
		formatter: NewFormatter(cfg.Format),
		output:    &StdOutput{},
	}
}

// NewExporterWithOutput はカスタム出力を使用するExporterを作成する
func NewExporterWithOutput(client backlog.Client, cfg *config.Config, output Output) *Exporter {
	return &Exporter{
		client:    client,
		config:    cfg,
		formatter: NewFormatter(cfg.Format),
		output:    output,
	}
}

// Run はエクスポート処理を実行する
func (e *Exporter) Run(ctx context.Context) (string, error) {
	// 1. プロジェクト情報を取得
	e.output.Printf("Connecting to %s.%s...\n", e.config.Space, e.config.Domain)

	project, err := e.client.GetProject(ctx, e.config.Project)
	if err != nil {
		return "", fmt.Errorf("failed to get project: %w", err)
	}

	e.output.Printf("Project: %s (%s)\n", project.ProjectKey, project.Name)

	// 2. 状態一覧を取得
	e.output.Printf("Fetching statuses... ")
	statuses, err := e.client.GetStatuses(ctx, e.config.Project)
	if err != nil {
		return "", fmt.Errorf("failed to get statuses: %w", err)
	}
	e.output.Printf("done\n")

	// 3. 完了以外の状態IDを特定
	incompleteStatusIDs := e.getIncompleteStatusIDs(statuses)

	// 4. 課題一覧を取得
	issues, err := e.client.GetIssues(ctx, project.ID, incompleteStatusIDs, e.config.Assignee, func(fetched, total int) {
		if total > 0 && fetched == total {
			e.output.Printf("Fetching issues... %d/%d (complete)\n", fetched, total)
		} else {
			e.output.Printf("Fetching issues... %d\n", fetched)
		}
	})
	if err != nil {
		return "", fmt.Errorf("failed to get issues: %w", err)
	}

	// 5. 親子関係を構造化
	e.output.Printf("Building hierarchy... ")
	hierarchicalIssues, summary := e.buildHierarchy(issues)
	e.output.Printf("done\n\n")

	// 6. サマリー表示
	e.output.Printf("Summary:\n")
	e.output.Printf("  Total issues: %d\n", summary.Total)
	e.output.Printf("  Parent issues: %d\n", summary.ParentIssues)
	e.output.Printf("  Child issues: %d\n\n", summary.ChildIssues)

	// 7. エクスポートデータを作成
	exportData := &backlog.ExportData{
		Project:    project,
		ExportedAt: time.Now(),
		Summary:    summary,
		Issues:     hierarchicalIssues,
	}

	// 8. フォーマットして出力
	content, err := e.formatter.Format(exportData)
	if err != nil {
		return "", fmt.Errorf("failed to format output: %w", err)
	}

	// 9. ファイルに保存
	filename := e.generateFilename(project.ProjectKey)
	outputPath := filepath.Join(e.config.Output, filename)

	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	e.output.Printf("Output: %s\n", outputPath)
	e.output.Printf("Done!\n")

	return outputPath, nil
}

// getIncompleteStatusIDs は完了以外の状態IDを取得する
func (e *Exporter) getIncompleteStatusIDs(statuses []*backlog.Status) []int {
	var ids []int
	for _, s := range statuses {
		// 「完了」を除外（デフォルトのID=4、または名前が「完了」のもの）
		if s.ID == defaultCompletedStatusID || s.Name == "完了" {
			continue
		}
		ids = append(ids, s.ID)
	}
	return ids
}

// buildHierarchy は課題一覧から親子階層を構築する
func (e *Exporter) buildHierarchy(issues []*backlog.Issue) ([]*backlog.HierarchicalIssue, backlog.ExportSummary) {
	// ID -> HierarchicalIssue のマップを作成
	issueMap := make(map[int]*backlog.HierarchicalIssue)
	for _, issue := range issues {
		issueMap[issue.ID] = &backlog.HierarchicalIssue{
			Issue:    issue,
			Children: make([]*backlog.HierarchicalIssue, 0),
		}
	}

	// 親子関係を構築
	var roots []*backlog.HierarchicalIssue
	childCount := 0

	for _, issue := range issues {
		hi := issueMap[issue.ID]
		if issue.ParentIssueID != nil {
			// 親課題がある場合
			if parent, ok := issueMap[*issue.ParentIssueID]; ok {
				parent.Children = append(parent.Children, hi)
				childCount++
			} else {
				// 親課題が取得対象に含まれていない場合はルートとして扱う
				roots = append(roots, hi)
			}
		} else {
			// 親課題がない場合はルートに追加
			roots = append(roots, hi)
		}
	}

	summary := backlog.ExportSummary{
		Total:        len(issues),
		ParentIssues: len(issues) - childCount,
		ChildIssues:  childCount,
	}

	return roots, summary
}

// generateFilename は出力ファイル名を生成する
func (e *Exporter) generateFilename(projectKey string) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_tasks_%s.%s", projectKey, timestamp, e.formatter.Extension())
}
