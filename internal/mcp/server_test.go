package mcp_test

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalmcp "github.com/daikw/ccpersona/internal/mcp"
	"github.com/daikw/ccpersona/internal/mcp/adapter"
)

func TestNewSpeakService_NotNil(t *testing.T) {
	synth := &mockSynthesizer{}
	player := &mockPlayer{}
	svc := internalmcp.NewSpeakService(synth, player)
	require.NotNil(t, svc)
}

func TestAdapterServer_Initialization(t *testing.T) {
	srv := adapter.NewServer("ccpersona-test", "v0.0.0-test")
	assert.NotNil(t, srv)
}

func TestAdapterServer_AddToolDoesNotPanic(t *testing.T) {
	srv := adapter.NewServer("ccpersona-test", "v0.0.0-test")

	assert.NotPanics(t, func() {
		srv.AddTool(
			"speak",
			"テスト用ツール",
			[]adapter.ToolParam{
				{Name: "text", Description: "テキスト", Type: "string", Required: true},
				{Name: "speaker", Description: "スピーカーID", Type: "number"},
			},
			func(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
				return mcplib.NewToolResultText("ok"), nil
			},
		)
	})
}
