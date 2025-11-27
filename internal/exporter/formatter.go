package exporter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/miyanaga/backlog-exporter/internal/backlog"
	"github.com/miyanaga/backlog-exporter/internal/config"
)

// Formatter は出力フォーマッターのインターフェース
type Formatter interface {
	Format(data *backlog.ExportData) ([]byte, error)
	Extension() string
}

// NewFormatter は指定されたフォーマットに対応するフォーマッターを作成する
func NewFormatter(format config.OutputFormat) Formatter {
	switch format {
	case config.FormatJSON:
		return &JSONFormatter{}
	case config.FormatMarkdown:
		return &MarkdownFormatter{}
	default:
		return &TXTFormatter{}
	}
}

// ============================================
// TXT Formatter
// ============================================

// TXTFormatter はテキスト形式のフォーマッター
type TXTFormatter struct{}

func (f *TXTFormatter) Extension() string {
	return "txt"
}

func (f *TXTFormatter) Format(data *backlog.ExportData) ([]byte, error) {
	var sb strings.Builder

	// ヘッダー
	sb.WriteString("================================================================================\n")
	sb.WriteString(fmt.Sprintf("プロジェクト: %s - %s\n", data.Project.ProjectKey, data.Project.Name))
	sb.WriteString(fmt.Sprintf("取得日時: %s\n", data.ExportedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("未完了タスク数: %d件（親課題: %d件、子課題: %d件）\n",
		data.Summary.Total, data.Summary.ParentIssues, data.Summary.ChildIssues))
	sb.WriteString("================================================================================\n\n")

	// 課題一覧
	for i, issue := range data.Issues {
		f.formatIssue(&sb, issue, false)

		if i < len(data.Issues)-1 {
			sb.WriteString("\n--------------------------------------------------------------------------------\n\n")
		}
	}

	return []byte(sb.String()), nil
}

func (f *TXTFormatter) formatIssue(sb *strings.Builder, hi *backlog.HierarchicalIssue, isChild bool) {
	issue := hi.Issue
	prefix := ""
	if isChild {
		prefix = "     "
	}

	sb.WriteString(fmt.Sprintf("%s[%s] %s\n", prefix, issue.IssueKey, issue.Summary))
	sb.WriteString(fmt.Sprintf("%s  状態: %s\n", prefix, f.getStatusName(issue)))
	sb.WriteString(fmt.Sprintf("%s  優先度: %s\n", prefix, f.getPriorityName(issue)))
	sb.WriteString(fmt.Sprintf("%s  担当者: %s\n", prefix, f.getAssigneeName(issue)))
	sb.WriteString(fmt.Sprintf("%s  期限日: %s\n", prefix, f.getDueDate(issue)))
	if !isChild {
		sb.WriteString(fmt.Sprintf("%s  作成日: %s\n", prefix, issue.Created.Format("2006-01-02")))
		sb.WriteString(fmt.Sprintf("%s  更新日: %s\n", prefix, issue.Updated.Format("2006-01-02")))
	}

	// 子課題
	if len(hi.Children) > 0 {
		sb.WriteString("\n")
		for i, child := range hi.Children {
			isLast := i == len(hi.Children)-1
			if isLast {
				sb.WriteString("  └─ ")
			} else {
				sb.WriteString("  ├─ ")
			}
			sb.WriteString(fmt.Sprintf("[%s] %s\n", child.Issue.IssueKey, child.Issue.Summary))

			linePrefix := "  │    "
			if isLast {
				linePrefix = "       "
			}
			sb.WriteString(fmt.Sprintf("%s状態: %s\n", linePrefix, f.getStatusName(child.Issue)))
			sb.WriteString(fmt.Sprintf("%s優先度: %s\n", linePrefix, f.getPriorityName(child.Issue)))
			sb.WriteString(fmt.Sprintf("%s担当者: %s\n", linePrefix, f.getAssigneeName(child.Issue)))
			sb.WriteString(fmt.Sprintf("%s期限日: %s\n", linePrefix, f.getDueDate(child.Issue)))

			if !isLast {
				sb.WriteString("  │\n")
			}
		}
	}
}

func (f *TXTFormatter) getStatusName(issue *backlog.Issue) string {
	if issue.Status != nil {
		return issue.Status.Name
	}
	return "-"
}

func (f *TXTFormatter) getPriorityName(issue *backlog.Issue) string {
	if issue.Priority != nil {
		return issue.Priority.Name
	}
	return "-"
}

func (f *TXTFormatter) getAssigneeName(issue *backlog.Issue) string {
	if issue.Assignee != nil {
		return issue.Assignee.Name
	}
	return "(未割当)"
}

func (f *TXTFormatter) getDueDate(issue *backlog.Issue) string {
	if issue.DueDate != nil && *issue.DueDate != "" {
		return *issue.DueDate
	}
	return "-"
}

// ============================================
// Markdown Formatter
// ============================================

// MarkdownFormatter はMarkdown形式のフォーマッター
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Extension() string {
	return "md"
}

func (f *MarkdownFormatter) Format(data *backlog.ExportData) ([]byte, error) {
	var sb strings.Builder

	// ヘッダー
	sb.WriteString(fmt.Sprintf("# %s - %s 未完了タスク一覧\n\n", data.Project.ProjectKey, data.Project.Name))
	sb.WriteString(fmt.Sprintf("> 取得日時: %s  \n", data.ExportedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("> 未完了タスク数: %d件（親課題: %d件、子課題: %d件）\n\n",
		data.Summary.Total, data.Summary.ParentIssues, data.Summary.ChildIssues))
	sb.WriteString("---\n\n")

	// 課題一覧
	for _, issue := range data.Issues {
		f.formatIssue(&sb, issue)
		sb.WriteString("\n---\n\n")
	}

	return []byte(sb.String()), nil
}

func (f *MarkdownFormatter) formatIssue(sb *strings.Builder, hi *backlog.HierarchicalIssue) {
	issue := hi.Issue

	sb.WriteString(fmt.Sprintf("## [%s] %s\n", issue.IssueKey, issue.Summary))
	sb.WriteString("| 項目 | 内容 |\n")
	sb.WriteString("|------|------|\n")
	sb.WriteString(fmt.Sprintf("| 状態 | %s |\n", f.getStatusName(issue)))
	sb.WriteString(fmt.Sprintf("| 優先度 | %s |\n", f.getPriorityName(issue)))
	sb.WriteString(fmt.Sprintf("| 担当者 | %s |\n", f.getAssigneeName(issue)))
	sb.WriteString(fmt.Sprintf("| 期限日 | %s |\n", f.getDueDate(issue)))
	sb.WriteString(fmt.Sprintf("| 作成日 | %s |\n", issue.Created.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("| 更新日 | %s |\n", issue.Updated.Format("2006-01-02")))

	// 子課題
	if len(hi.Children) > 0 {
		sb.WriteString("\n### 子課題\n\n")
		for _, child := range hi.Children {
			sb.WriteString(fmt.Sprintf("#### [%s] %s\n", child.Issue.IssueKey, child.Issue.Summary))
			sb.WriteString("| 項目 | 内容 |\n")
			sb.WriteString("|------|------|\n")
			sb.WriteString(fmt.Sprintf("| 状態 | %s |\n", f.getStatusName(child.Issue)))
			sb.WriteString(fmt.Sprintf("| 優先度 | %s |\n", f.getPriorityName(child.Issue)))
			sb.WriteString(fmt.Sprintf("| 担当者 | %s |\n", f.getAssigneeName(child.Issue)))
			sb.WriteString(fmt.Sprintf("| 期限日 | %s |\n", f.getDueDate(child.Issue)))
			sb.WriteString("\n")
		}
	}
}

func (f *MarkdownFormatter) getStatusName(issue *backlog.Issue) string {
	if issue.Status != nil {
		return issue.Status.Name
	}
	return "-"
}

func (f *MarkdownFormatter) getPriorityName(issue *backlog.Issue) string {
	if issue.Priority != nil {
		return issue.Priority.Name
	}
	return "-"
}

func (f *MarkdownFormatter) getAssigneeName(issue *backlog.Issue) string {
	if issue.Assignee != nil {
		return issue.Assignee.Name
	}
	return "-"
}

func (f *MarkdownFormatter) getDueDate(issue *backlog.Issue) string {
	if issue.DueDate != nil && *issue.DueDate != "" {
		return *issue.DueDate
	}
	return "-"
}

// ============================================
// JSON Formatter
// ============================================

// JSONFormatter はJSON形式のフォーマッター
type JSONFormatter struct{}

func (f *JSONFormatter) Extension() string {
	return "json"
}

// jsonExportData はJSON出力用のデータ構造
type jsonExportData struct {
	Project    jsonProject    `json:"project"`
	ExportedAt string         `json:"exportedAt"`
	Summary    jsonSummary    `json:"summary"`
	Issues     []jsonIssue    `json:"issues"`
}

type jsonProject struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type jsonSummary struct {
	Total        int `json:"total"`
	ParentIssues int `json:"parentIssues"`
	ChildIssues  int `json:"childIssues"`
}

type jsonIssue struct {
	ID        int         `json:"id"`
	IssueKey  string      `json:"issueKey"`
	Summary   string      `json:"summary"`
	Status    string      `json:"status"`
	Priority  string      `json:"priority"`
	Assignee  *string     `json:"assignee"`
	DueDate   *string     `json:"dueDate"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
	Children  []jsonIssue `json:"children"`
}

func (f *JSONFormatter) Format(data *backlog.ExportData) ([]byte, error) {
	output := jsonExportData{
		Project: jsonProject{
			ID:   data.Project.ID,
			Key:  data.Project.ProjectKey,
			Name: data.Project.Name,
		},
		ExportedAt: data.ExportedAt.Format(time.RFC3339),
		Summary: jsonSummary{
			Total:        data.Summary.Total,
			ParentIssues: data.Summary.ParentIssues,
			ChildIssues:  data.Summary.ChildIssues,
		},
		Issues: make([]jsonIssue, 0, len(data.Issues)),
	}

	for _, hi := range data.Issues {
		output.Issues = append(output.Issues, f.convertIssue(hi))
	}

	return json.MarshalIndent(output, "", "  ")
}

func (f *JSONFormatter) convertIssue(hi *backlog.HierarchicalIssue) jsonIssue {
	issue := hi.Issue
	ji := jsonIssue{
		ID:        issue.ID,
		IssueKey:  issue.IssueKey,
		Summary:   issue.Summary,
		Status:    f.getStatusName(issue),
		Priority:  f.getPriorityName(issue),
		Assignee:  f.getAssigneeName(issue),
		DueDate:   issue.DueDate,
		CreatedAt: issue.Created.Format(time.RFC3339),
		UpdatedAt: issue.Updated.Format(time.RFC3339),
		Children:  make([]jsonIssue, 0, len(hi.Children)),
	}

	for _, child := range hi.Children {
		ji.Children = append(ji.Children, f.convertIssue(child))
	}

	return ji
}

func (f *JSONFormatter) getStatusName(issue *backlog.Issue) string {
	if issue.Status != nil {
		return issue.Status.Name
	}
	return ""
}

func (f *JSONFormatter) getPriorityName(issue *backlog.Issue) string {
	if issue.Priority != nil {
		return issue.Priority.Name
	}
	return ""
}

func (f *JSONFormatter) getAssigneeName(issue *backlog.Issue) *string {
	if issue.Assignee != nil {
		name := issue.Assignee.Name
		return &name
	}
	return nil
}
