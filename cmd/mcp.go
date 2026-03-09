package main

import (
	"context"

	"github.com/daikw/ccpersona/internal/mcp"
	"github.com/urfave/cli/v3"
)

func handleMCP(ctx context.Context, c *cli.Command) error {
	version := c.Root().Version
	return mcp.RunServer(ctx, version)
}
