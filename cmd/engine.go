package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/daikw/ccpersona/internal/engine"
	"github.com/urfave/cli/v3"
)

// ANSI color helpers (disabled when stdout is not a terminal)
var (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func init() {
	if fi, err := os.Stdout.Stat(); err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		colorReset = ""
		colorGreen = ""
		colorYellow = ""
		colorRed = ""
		colorCyan = ""
		colorBold = ""
		colorDim = ""
	}
}

func parseTarget(c *cli.Command) ([]engine.EngineType, error) {
	target := c.Args().First()
	if target == "" {
		target = "all"
	}
	return engine.ParseEngineType(target)
}

func handleEngineInstall(ctx context.Context, c *cli.Command) error {
	types, err := parseTarget(c)
	if err != nil {
		return err
	}

	mgr, err := engine.NewServiceManager()
	if err != nil {
		return err
	}

	var errs []error
	for _, t := range types {
		info, err := engine.DiscoverEngine(t)
		if err != nil {
			fmt.Printf("  %s%s%s: binary not found (install the app first)\n", colorRed, t, colorReset)
			errs = append(errs, fmt.Errorf("%s: %w", t, err))
			continue
		}
		fmt.Printf("  %s%s%s: binary found -> %s\n", colorCyan, t, colorReset, info.BinaryPath)

		if err := mgr.Install(info); err != nil {
			fmt.Printf("  %s%s%s: %sinstall failed%s -> %v\n", colorCyan, t, colorReset, colorRed, colorReset, err)
			errs = append(errs, fmt.Errorf("%s install: %w", t, err))
			continue
		}
		fmt.Printf("  %s%s%s: %sservice installed%s\n", colorCyan, t, colorReset, colorGreen, colorReset)
	}
	return errors.Join(errs...)
}

func handleEngineUninstall(ctx context.Context, c *cli.Command) error {
	types, err := parseTarget(c)
	if err != nil {
		return err
	}

	mgr, err := engine.NewServiceManager()
	if err != nil {
		return err
	}

	var errs []error
	for _, t := range types {
		if err := mgr.Uninstall(t); err != nil {
			fmt.Printf("  %s%s%s: %suninstall failed%s -> %v\n", colorCyan, t, colorReset, colorRed, colorReset, err)
			errs = append(errs, fmt.Errorf("%s uninstall: %w", t, err))
			continue
		}
		fmt.Printf("  %s%s%s: %sservice uninstalled%s\n", colorCyan, t, colorReset, colorGreen, colorReset)
	}
	return errors.Join(errs...)
}

func handleEngineStart(ctx context.Context, c *cli.Command) error {
	types, err := parseTarget(c)
	if err != nil {
		return err
	}

	mgr, err := engine.NewServiceManager()
	if err != nil {
		return err
	}

	var errs []error
	for _, t := range types {
		if err := mgr.Start(t); err != nil {
			fmt.Printf("  %s%s%s: %sstart failed%s -> %v\n", colorCyan, t, colorReset, colorRed, colorReset, err)
			errs = append(errs, fmt.Errorf("%s start: %w", t, err))
			continue
		}
		fmt.Printf("  %s%s%s: %sstarted%s\n", colorCyan, t, colorReset, colorGreen, colorReset)
	}
	return errors.Join(errs...)
}

func handleEngineStop(ctx context.Context, c *cli.Command) error {
	types, err := parseTarget(c)
	if err != nil {
		return err
	}

	mgr, err := engine.NewServiceManager()
	if err != nil {
		return err
	}

	var errs []error
	for _, t := range types {
		if err := mgr.Stop(t); err != nil {
			fmt.Printf("  %s%s%s: %sstop failed%s -> %v\n", colorCyan, t, colorReset, colorRed, colorReset, err)
			errs = append(errs, fmt.Errorf("%s stop: %w", t, err))
			continue
		}
		fmt.Printf("  %s%s%s: %sstopped%s\n", colorCyan, t, colorReset, colorYellow, colorReset)
	}
	return errors.Join(errs...)
}

func handleEngineStatus(ctx context.Context, c *cli.Command) error {
	mgr, err := engine.NewServiceManager()
	if err != nil {
		return err
	}

	for _, t := range engine.AllEngineTypes() {
		status, err := mgr.Status(t)
		if err != nil {
			fmt.Printf("  %s%s%s: failed to get status -> %v\n", colorRed, t, colorReset, err)
			continue
		}

		info, discoverErr := engine.DiscoverEngine(t)
		binaryStatus := fmt.Sprintf("%snot found%s", colorRed, colorReset)
		if discoverErr == nil {
			binaryStatus = fmt.Sprintf("%s%s%s", colorDim, info.BinaryPath, colorReset)
		}

		installMark := fmt.Sprintf("%snot installed%s", colorYellow, colorReset)
		if status.Installed {
			installMark = fmt.Sprintf("%sinstalled%s", colorGreen, colorReset)
		}

		runMark := fmt.Sprintf("%sstopped%s", colorYellow, colorReset)
		if status.Running {
			runMark = fmt.Sprintf("%srunning%s (PID: %d)", colorGreen, colorReset, status.PID)
		}

		fmt.Printf("  %s%s%s%s:\n", colorBold, colorCyan, t, colorReset)
		fmt.Printf("    binary:  %s\n", binaryStatus)
		fmt.Printf("    service: %s %s[%s]%s\n", installMark, colorDim, status.Label, colorReset)
		fmt.Printf("    status:  %s\n", runMark)
	}
	return nil
}
