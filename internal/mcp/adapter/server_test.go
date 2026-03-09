package adapter_test

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/daikw/ccpersona/internal/mcp/adapter"
)

func TestNewServer_NotNil(t *testing.T) {
	srv := adapter.NewServer("test", "0.0.1")
	require.NotNil(t, srv)
}

func TestAddTool_StringAndNumber(t *testing.T) {
	srv := adapter.NewServer("test", "0.0.1")

	assert.NotPanics(t, func() {
		srv.AddTool(
			"speak",
			"read aloud",
			[]adapter.ToolParam{
				{Name: "text", Description: "text to speak", Type: "string", Required: true},
				{Name: "speaker", Description: "speaker id", Type: "number"},
				{Name: "provider", Description: "provider name", Type: "string"},
			},
			func(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
				return mcplib.NewToolResultText("done"), nil
			},
		)
	})
}

func TestAddTool_NoParams(t *testing.T) {
	srv := adapter.NewServer("test", "0.0.1")

	assert.NotPanics(t, func() {
		srv.AddTool(
			"ping",
			"ping the server",
			nil,
			func(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
				return mcplib.NewToolResultText("pong"), nil
			},
		)
	})
}
