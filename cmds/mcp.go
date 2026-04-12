package cmds

import (
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	xfmcp "github.com/sttts/xf-cli/mcp"
)

type MCPCommand struct{}

func (cmd *MCPCommand) Run(app *App) error {
	cfg, err := app.mcpConfig()
	if err != nil {
		return err
	}

	mcpServer, err := xfmcp.NewServer(cfg)
	if err != nil {
		return err
	}

	logger := log.New(os.Stderr, "mcp: ", log.LstdFlags)
	return server.ServeStdio(mcpServer, server.WithErrorLogger(logger))
}
