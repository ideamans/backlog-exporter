package backlog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	maxCount       = 100 // APIの最大取得件数
)

// HTTPClient は HTTP リクエストを行うインターフェース（テスト用）
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// APIClient は Backlog API クライアントの実装
type APIClient struct {
	baseURL    string
	apiKey     string
	httpClient HTTPClient
}

// NewClient は新しい Backlog API クライアントを作成する
func NewClient(space, domain, apiKey string) *APIClient {
	return &APIClient{
		baseURL:    fmt.Sprintf("https://%s.%s/api/v2", space, domain),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// NewClientWithHTTPClient はカスタムHTTPクライアントを使用する Backlog API クライアントを作成する
func NewClientWithHTTPClient(baseURL, apiKey string, httpClient HTTPClient) *APIClient {
	return &APIClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// GetProject はプロジェクト情報を取得する
func (c *APIClient) GetProject(ctx context.Context, projectIDOrKey string) (*Project, error) {
	endpoint := fmt.Sprintf("%s/projects/%s", c.baseURL, url.PathEscape(projectIDOrKey))

	var project Project
	if err := c.doRequest(ctx, endpoint, nil, &project); err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

// GetStatuses はプロジェクトの状態一覧を取得する
func (c *APIClient) GetStatuses(ctx context.Context, projectIDOrKey string) ([]*Status, error) {
	endpoint := fmt.Sprintf("%s/projects/%s/statuses", c.baseURL, url.PathEscape(projectIDOrKey))

	var statuses []*Status
	if err := c.doRequest(ctx, endpoint, nil, &statuses); err != nil {
		return nil, fmt.Errorf("failed to get statuses: %w", err)
	}

	return statuses, nil
}

// GetIssues は課題一覧を取得する（ページネーション処理済み）
func (c *APIClient) GetIssues(ctx context.Context, projectID int, statusIDs []int, assigneeID *int, progressFn func(fetched, total int)) ([]*Issue, error) {
	var allIssues []*Issue
	offset := 0

	for {
		params := url.Values{}
		params.Set("projectId[]", strconv.Itoa(projectID))
		params.Set("count", strconv.Itoa(maxCount))
		params.Set("offset", strconv.Itoa(offset))
		params.Set("sort", "created")
		params.Set("order", "asc")

		for _, statusID := range statusIDs {
			params.Add("statusId[]", strconv.Itoa(statusID))
		}

		if assigneeID != nil {
			params.Set("assigneeId[]", strconv.Itoa(*assigneeID))
		}

		endpoint := fmt.Sprintf("%s/issues", c.baseURL)

		var issues []*Issue
		if err := c.doRequest(ctx, endpoint, params, &issues); err != nil {
			return nil, fmt.Errorf("failed to get issues: %w", err)
		}

		allIssues = append(allIssues, issues...)

		// 進捗通知
		if progressFn != nil {
			progressFn(len(allIssues), -1) // 総数は不明なので-1
		}

		// ページネーション: 取得件数がmaxCount未満なら終了
		if len(issues) < maxCount {
			break
		}

		offset += maxCount
	}

	// 最終的な進捗通知
	if progressFn != nil {
		progressFn(len(allIssues), len(allIssues))
	}

	return allIssues, nil
}

// doRequest は API リクエストを実行する
func (c *APIClient) doRequest(ctx context.Context, endpoint string, params url.Values, result interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("apiKey", c.apiKey)

	fullURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && len(apiErr.Errors) > 0 {
			return &apiErr
		}
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}
