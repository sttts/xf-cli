package main

import (
	"context"
	"testing"

	mcpapi "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	xfmcp "github.com/sttts/xf-cli/mcp"
)

func newLiveMCPServer(t *testing.T) *mcptest.Server {
	t.Helper()

	username, password := requireLiveCredentials(t)
	tools, err := xfmcp.Tools(xfmcp.Config{
		BaseURL:  "https://www.rc-network.de",
		Username: username,
		Password: password,
	})
	if err != nil {
		t.Fatalf("build mcp tools: %v", err)
	}

	srv, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("start mcp test server: %v", err)
	}
	t.Cleanup(srv.Close)

	return srv
}

func TestLiveMCPListTools(t *testing.T) {
	srv := newLiveMCPServer(t)

	result, err := srv.Client().ListTools(context.Background(), mcpapi.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	expected := map[string]bool{
		"list_forums":                  false,
		"list_threads":                 false,
		"read_thread":                  false,
		"search_threads":               false,
		"search_posts":                 false,
		"read_profile":                 false,
		"list_user_posts":              false,
		"list_user_threads":            false,
		"list_my_threads":              false,
		"list_threads_i_participated":  false,
		"follow_link":                  false,
		"get_image":                    false,
	}

	for _, tool := range result.Tools {
		if _, ok := expected[tool.Name]; ok {
			expected[tool.Name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Fatalf("expected MCP tool %q to be registered", name)
		}
	}
}

func TestLiveMCPReadThread(t *testing.T) {
	srv := newLiveMCPServer(t)

	var req mcpapi.CallToolRequest
	req.Params.Name = "read_thread"
	req.Params.Arguments = map[string]any{
		"thread_url": "/threads/eure-sch%C3%B6nsten-modelle.144946/",
	}

	result, err := srv.Client().CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful tool result, got error: %+v", result.Content)
	}

	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content, got %T", result.StructuredContent)
	}
	if structured["title"] == "" {
		t.Fatal("expected thread title in structured content")
	}

	posts, ok := structured["posts"].([]any)
	if !ok || len(posts) == 0 {
		t.Fatal("expected posts in structured content")
	}
}
