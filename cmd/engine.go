package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/daikw/ccpersona/internal/cliui"
	"github.com/daikw/ccpersona/internal/engine"
	"github.com/urfave/cli/v3"
)

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
			fmt.Printf("  %s: binary not found (install the app first)\n", cliui.Failure(t))
			errs = append(errs, fmt.Errorf("%s: %w", t, err))
			continue
		}
		fmt.Printf("  %s: binary found -> %s\n", cliui.Label(t), info.BinaryPath)

		if err := mgr.Install(info); err != nil {
			fmt.Printf("  %s: %s -> %v\n", cliui.Label(t), cliui.Failure("install failed"), err)
			errs = append(errs, fmt.Errorf("%s install: %w", t, err))
			continue
		}
		fmt.Printf("  %s: %s\n", cliui.Label(t), cliui.Success("service installed"))
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
			fmt.Printf("  %s: %s -> %v\n", cliui.Label(t), cliui.Failure("uninstall failed"), err)
			errs = append(errs, fmt.Errorf("%s uninstall: %w", t, err))
			continue
		}
		fmt.Printf("  %s: %s\n", cliui.Label(t), cliui.Success("service uninstalled"))
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
			fmt.Printf("  %s: %s -> %v\n", cliui.Label(t), cliui.Failure("start failed"), err)
			errs = append(errs, fmt.Errorf("%s start: %w", t, err))
			continue
		}
		fmt.Printf("  %s: %s\n", cliui.Label(t), cliui.Success("started"))
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
			fmt.Printf("  %s: %s -> %v\n", cliui.Label(t), cliui.Failure("stop failed"), err)
			errs = append(errs, fmt.Errorf("%s stop: %w", t, err))
			continue
		}
		fmt.Printf("  %s: %s\n", cliui.Label(t), cliui.Warn("stopped"))
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
			fmt.Printf("  %s: failed to get status -> %v\n", cliui.Failure(t), err)
			continue
		}

		info, discoverErr := engine.DiscoverEngine(t)
		binaryStatus := cliui.Failure("not found")
		if discoverErr == nil {
			binaryStatus = cliui.Muted(info.BinaryPath)
		}

		installMark := cliui.Warn("not installed")
		if status.Installed {
			installMark = cliui.Success("installed")
		}

		runMark := cliui.Warn("stopped")
		if status.Running {
			runMark = fmt.Sprintf("%s (PID: %d)", cliui.Success("running"), status.PID)
		}

		fmt.Printf("  %s:\n", cliui.Engine(t))
		fmt.Printf("    binary:  %s\n", binaryStatus)
		fmt.Printf("    service: %s %s\n", installMark, cliui.Muted("["+status.Label+"]"))
		fmt.Printf("    status:  %s\n", runMark)
	}
	return nil
}
