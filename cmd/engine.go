package main

import (
	"context"
	"errors"
	"fmt"

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
			fmt.Printf("  %s: %v\n", t, err)
			errs = append(errs, fmt.Errorf("%s: %w", t, err))
			continue
		}
		fmt.Printf("  %s: binary found -> %s\n", t, info.BinaryPath)

		if err := mgr.Install(info); err != nil {
			fmt.Printf("  %s: install failed -> %v\n", t, err)
			errs = append(errs, fmt.Errorf("%s install: %w", t, err))
			continue
		}
		fmt.Printf("  %s: service installed\n", t)
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
			fmt.Printf("  %s: uninstall failed -> %v\n", t, err)
			errs = append(errs, fmt.Errorf("%s uninstall: %w", t, err))
			continue
		}
		fmt.Printf("  %s: service uninstalled\n", t)
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
			fmt.Printf("  %s: start failed -> %v\n", t, err)
			errs = append(errs, fmt.Errorf("%s start: %w", t, err))
			continue
		}
		fmt.Printf("  %s: started\n", t)
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
			fmt.Printf("  %s: stop failed -> %v\n", t, err)
			errs = append(errs, fmt.Errorf("%s stop: %w", t, err))
			continue
		}
		fmt.Printf("  %s: stopped\n", t)
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
			fmt.Printf("  %s: failed to get status -> %v\n", t, err)
			continue
		}

		info, discoverErr := engine.DiscoverEngine(t)
		binaryStatus := "not found"
		if discoverErr == nil {
			binaryStatus = info.BinaryPath
		}

		installMark := "not installed"
		if status.Installed {
			installMark = "installed"
		}

		runMark := "stopped"
		if status.Running {
			runMark = fmt.Sprintf("running (PID: %d)", status.PID)
		}

		fmt.Printf("  %s:\n", t)
		fmt.Printf("    binary:  %s\n", binaryStatus)
		fmt.Printf("    service: %s [%s]\n", installMark, status.Label)
		fmt.Printf("    status:  %s\n", runMark)
	}
	return nil
}
