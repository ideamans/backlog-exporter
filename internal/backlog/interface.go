package backlog

import "context"

// Client はBacklog APIクライアントのインターフェース
type Client interface {
	// GetProject はプロジェクト情報を取得する
	GetProject(ctx context.Context, projectIDOrKey string) (*Project, error)

	// GetStatuses はプロジェクトの状態一覧を取得する
	GetStatuses(ctx context.Context, projectIDOrKey string) ([]*Status, error)

	// GetIssues は課題一覧を取得する（ページネーション処理済み）
	// statusIDs に指定した状態の課題のみ取得
	// assigneeID が指定された場合は担当者でフィルタリング
	// progressFn は進捗状況を通知するコールバック（nil可）
	GetIssues(ctx context.Context, projectID int, statusIDs []int, assigneeID *int, progressFn func(fetched, total int)) ([]*Issue, error)
}

// ProgressCallback は進捗を通知するコールバック関数の型
type ProgressCallback func(fetched, total int)
