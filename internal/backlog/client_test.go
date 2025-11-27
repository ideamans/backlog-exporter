package backlog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIClient_GetProject(t *testing.T) {
	// モックサーバーを作成
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/projects/MYPROJ" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("apiKey") != "test-api-key" {
			t.Errorf("unexpected apiKey: %s", r.URL.Query().Get("apiKey"))
		}

		project := Project{
			ID:         1,
			ProjectKey: "MYPROJ",
			Name:       "マイプロジェクト",
		}
		json.NewEncoder(w).Encode(project)
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL+"/api/v2", "test-api-key", server.Client())

	ctx := context.Background()
	project, err := client.GetProject(ctx, "MYPROJ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if project.ID != 1 {
		t.Errorf("expected project ID 1, got %d", project.ID)
	}
	if project.ProjectKey != "MYPROJ" {
		t.Errorf("expected project key MYPROJ, got %s", project.ProjectKey)
	}
	if project.Name != "マイプロジェクト" {
		t.Errorf("expected project name マイプロジェクト, got %s", project.Name)
	}
}

func TestAPIClient_GetStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/projects/MYPROJ/statuses" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		statuses := []*Status{
			{ID: 1, Name: "未対応"},
			{ID: 2, Name: "処理中"},
			{ID: 3, Name: "処理済み"},
			{ID: 4, Name: "完了"},
		}
		json.NewEncoder(w).Encode(statuses)
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL+"/api/v2", "test-api-key", server.Client())

	ctx := context.Background()
	statuses, err := client.GetStatuses(ctx, "MYPROJ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(statuses) != 4 {
		t.Errorf("expected 4 statuses, got %d", len(statuses))
	}
	if statuses[0].Name != "未対応" {
		t.Errorf("expected first status to be 未対応, got %s", statuses[0].Name)
	}
}

func TestAPIClient_GetIssues(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		requestCount++
		var issues []*Issue

		// 最初のリクエストでは50件を返す（ページネーション終了）
		for i := 0; i < 50; i++ {
			issues = append(issues, &Issue{
				ID:       i + 1,
				IssueKey: "MYPROJ-" + string(rune('1'+i)),
				Summary:  "Test Issue",
			})
		}

		json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL+"/api/v2", "test-api-key", server.Client())

	ctx := context.Background()
	var progressCalls int
	issues, err := client.GetIssues(ctx, 1, []int{1, 2, 3}, nil, func(fetched, total int) {
		progressCalls++
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 50 {
		t.Errorf("expected 50 issues, got %d", len(issues))
	}
	if requestCount != 1 {
		t.Errorf("expected 1 request, got %d", requestCount)
	}
	if progressCalls < 1 {
		t.Errorf("expected at least 1 progress callback, got %d", progressCalls)
	}
}

func TestAPIClient_GetIssues_Pagination(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var issues []*Issue

		// 最初のリクエストでは100件、2回目は50件を返す
		count := 100
		if requestCount > 1 {
			count = 50
		}

		for i := 0; i < count; i++ {
			issues = append(issues, &Issue{
				ID:       (requestCount-1)*100 + i + 1,
				IssueKey: "MYPROJ-" + string(rune('0'+requestCount)) + string(rune('0'+i%10)),
				Summary:  "Test Issue",
			})
		}

		json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(server.URL+"/api/v2", "test-api-key", server.Client())

	ctx := context.Background()
	issues, err := client.GetIssues(ctx, 1, []int{1, 2, 3}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 150 {
		t.Errorf("expected 150 issues, got %d", len(issues))
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount)
	}
}

func TestAPIClient_GetProject_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		apiErr := APIError{
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

	client := NewClientWithHTTPClient(server.URL+"/api/v2", "test-api-key", server.Client())

	ctx := context.Background()
	_, err := client.GetProject(ctx, "NOTFOUND")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
