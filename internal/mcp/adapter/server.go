// Package adapter wraps mark3labs/mcp-go, isolating direct library usage to this layer.
package adapter

import (
	"context"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolHandler is a function that handles a tool call.
type ToolHandler func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

// ToolParam describes a single parameter for a tool.
type ToolParam struct {
	Name        string
	Description string
	Required    bool
	// Type is "string" or "number"
	Type string
}

// Server wraps an MCP server instance.
type Server struct {
	mcpServer *server.MCPServer
}

// NewServer creates a new MCP server with the given name and version.
func NewServer(name, version string) *Server {
	s := server.NewMCPServer(name, version)
	return &Server{mcpServer: s}
}

// AddTool registers a tool with the server.
func (s *Server) AddTool(name, description string, params []ToolParam, handler ToolHandler) {
	opts := []mcp.ToolOption{
		mcp.WithDescription(description),
	}

	for _, p := range params {
		propertyOpts := []mcp.PropertyOption{
			mcp.Description(p.Description),
		}
		if p.Required {
			propertyOpts = append(propertyOpts, mcp.Required())
		}

		switch p.Type {
		case "number":
			opts = append(opts, mcp.WithNumber(p.Name, propertyOpts...))
		default:
			opts = append(opts, mcp.WithString(p.Name, propertyOpts...))
		}
	}

	tool := mcp.NewTool(name, opts...)
	s.mcpServer.AddTool(tool, server.ToolHandlerFunc(handler))
}

// ServeStdio starts the MCP server using stdin/stdout transport.
// Logs must be written to stderr by callers to avoid polluting the MCP protocol stream.
func (s *Server) ServeStdio(ctx context.Context) error {
	stdioServer := server.NewStdioServer(s.mcpServer)
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}
