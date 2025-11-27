package backlog

import "context"

// MockClient はテスト用のモッククライアント
type MockClient struct {
	GetProjectFunc   func(ctx context.Context, projectIDOrKey string) (*Project, error)
	GetStatusesFunc  func(ctx context.Context, projectIDOrKey string) ([]*Status, error)
	GetIssuesFunc    func(ctx context.Context, projectID int, statusIDs []int, assigneeID *int, progressFn func(fetched, total int)) ([]*Issue, error)
}

// GetProject はモック実装
func (m *MockClient) GetProject(ctx context.Context, projectIDOrKey string) (*Project, error) {
	if m.GetProjectFunc != nil {
		return m.GetProjectFunc(ctx, projectIDOrKey)
	}
	return nil, nil
}

// GetStatuses はモック実装
func (m *MockClient) GetStatuses(ctx context.Context, projectIDOrKey string) ([]*Status, error) {
	if m.GetStatusesFunc != nil {
		return m.GetStatusesFunc(ctx, projectIDOrKey)
	}
	return nil, nil
}

// GetIssues はモック実装
func (m *MockClient) GetIssues(ctx context.Context, projectID int, statusIDs []int, assigneeID *int, progressFn func(fetched, total int)) ([]*Issue, error) {
	if m.GetIssuesFunc != nil {
		return m.GetIssuesFunc(ctx, projectID, statusIDs, assigneeID, progressFn)
	}
	return nil, nil
}
