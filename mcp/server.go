package xfmcp

import (
	"fmt"
	"strings"
	"sync"

	mcpapi "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sttts/xf-cli/auth"
)

type Config struct {
	BaseURL  string
	Username string
	Password string
	Verbose  bool
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("base URL is required")
	}
	if strings.TrimSpace(c.Username) == "" {
		return fmt.Errorf("MCP mode requires XF_USERNAME or --username")
	}
	if strings.TrimSpace(c.Password) == "" {
		return fmt.Errorf("MCP mode requires XF_PASSWORD or --password")
	}

	return nil
}

type SessionProvider struct {
	config Config

	mu      sync.Mutex
	client  *auth.Client
	session auth.SessionInfo
	ready   bool
}

func NewSessionProvider(config Config) (*SessionProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &SessionProvider{config: config}, nil
}

func (p *SessionProvider) Login() (*auth.Client, auth.SessionInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client == nil {
		client, err := auth.NewClient(p.config.BaseURL, p.config.Verbose)
		if err != nil {
			return nil, auth.SessionInfo{}, err
		}
		p.client = client
	}

	if !p.ready {
		session, err := p.client.Login(p.config.Username, p.config.Password)
		if err != nil {
			return nil, auth.SessionInfo{}, err
		}
		p.session = session
		p.ready = true
	}

	return p.client, p.session, nil
}

func NewServer(config Config) (*server.MCPServer, error) {
	tools, err := Tools(config)
	if err != nil {
		return nil, err
	}

	mcpServer := server.NewMCPServer(
		"xf-cli",
		"0.1.0",
		server.WithInstructions("Read-only XenForo frontend access for rc-network.de over browser login. Use the provided tools instead of direct scraping."),
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	mcpServer.AddTools(tools...)
	return mcpServer, nil
}

func readOnlyTool(name, description string, opts ...mcpapi.ToolOption) mcpapi.Tool {
	toolOpts := []mcpapi.ToolOption{
		mcpapi.WithDescription(description),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	}
	toolOpts = append(toolOpts, opts...)

	return mcpapi.NewTool(name, toolOpts...)
}
