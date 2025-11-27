package backlog

import "time"

// Project はBacklogプロジェクトを表す
type Project struct {
	ID              int    `json:"id"`
	ProjectKey      string `json:"projectKey"`
	Name            string `json:"name"`
	ChartEnabled    bool   `json:"chartEnabled"`
	SubtaskingEnabled bool `json:"subtaskingEnabled"`
	TextFormattingRule string `json:"textFormattingRule"`
}

// Status は課題の状態を表す
type Status struct {
	ID           int    `json:"id"`
	ProjectID    int    `json:"projectId"`
	Name         string `json:"name"`
	Color        string `json:"color"`
	DisplayOrder int    `json:"displayOrder"`
}

// Priority は課題の優先度を表す
type Priority struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// User はBacklogユーザーを表す
type User struct {
	ID          int    `json:"id"`
	UserID      string `json:"userId"`
	Name        string `json:"name"`
	RoleType    int    `json:"roleType"`
	Lang        string `json:"lang"`
	MailAddress string `json:"mailAddress"`
}

// IssueType は課題種別を表す
type IssueType struct {
	ID           int    `json:"id"`
	ProjectID    int    `json:"projectId"`
	Name         string `json:"name"`
	Color        string `json:"color"`
	DisplayOrder int    `json:"displayOrder"`
}

// Issue はBacklog課題を表す
type Issue struct {
	ID             int        `json:"id"`
	ProjectID      int        `json:"projectId"`
	IssueKey       string     `json:"issueKey"`
	KeyID          int        `json:"keyId"`
	IssueType      *IssueType `json:"issueType"`
	Summary        string     `json:"summary"`
	Description    string     `json:"description"`
	Priority       *Priority  `json:"priority"`
	Status         *Status    `json:"status"`
	Assignee       *User      `json:"assignee"`
	StartDate      *string    `json:"startDate"`
	DueDate        *string    `json:"dueDate"`
	EstimatedHours *float64   `json:"estimatedHours"`
	ActualHours    *float64   `json:"actualHours"`
	ParentIssueID  *int       `json:"parentIssueId"`
	CreatedUser    *User      `json:"createdUser"`
	Created        time.Time  `json:"created"`
	UpdatedUser    *User      `json:"updatedUser"`
	Updated        time.Time  `json:"updated"`
}

// HierarchicalIssue は親子関係を持つ課題を表す
type HierarchicalIssue struct {
	Issue    *Issue
	Children []*HierarchicalIssue
}

// ExportData はエクスポートデータを表す
type ExportData struct {
	Project    *Project
	ExportedAt time.Time
	Summary    ExportSummary
	Issues     []*HierarchicalIssue
}

// ExportSummary はエクスポートのサマリーを表す
type ExportSummary struct {
	Total        int `json:"total"`
	ParentIssues int `json:"parentIssues"`
	ChildIssues  int `json:"childIssues"`
}

// APIError はBacklog APIからのエラーレスポンスを表す
type APIError struct {
	Errors []struct {
		Message  string `json:"message"`
		Code     int    `json:"code"`
		MoreInfo string `json:"moreInfo"`
	} `json:"errors"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return e.Errors[0].Message
	}
	return "unknown API error"
}
