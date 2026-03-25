// Package cliui provides shared terminal color styles for CLI output.
package cliui

import "github.com/fatih/color"

var (
	// Engine highlights engine/service names.
	Engine = color.New(color.FgCyan, color.Bold).SprintFunc()

	// Success indicates positive outcomes (installed, running, connected).
	Success = color.New(color.FgGreen).SprintFunc()

	// Warn indicates intermediate states (stopped, not installed).
	Warn = color.New(color.FgYellow).SprintFunc()

	// Failure indicates errors or missing resources.
	Failure = color.New(color.FgRed).SprintFunc()

	// Muted renders de-emphasized text (paths, labels, metadata).
	Muted = color.New(color.Faint).SprintFunc()

	// Header renders section headers.
	Header = color.New(color.Bold).SprintFunc()

	// Label renders key names in key-value output.
	Label = color.New(color.FgCyan).SprintFunc()
)
