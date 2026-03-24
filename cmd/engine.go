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
		fmt.Printf("  %s: バイナリ検出 → %s\n", t, info.BinaryPath)

		if err := mgr.Install(info); err != nil {
			fmt.Printf("  %s: インストール失敗 → %v\n", t, err)
			errs = append(errs, fmt.Errorf("%s install: %w", t, err))
			continue
		}
		fmt.Printf("  %s: サービスインストール完了\n", t)
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
			fmt.Printf("  %s: アンインストール失敗 → %v\n", t, err)
			errs = append(errs, fmt.Errorf("%s uninstall: %w", t, err))
			continue
		}
		fmt.Printf("  %s: サービスアンインストール完了\n", t)
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
			fmt.Printf("  %s: 起動失敗 → %v\n", t, err)
			errs = append(errs, fmt.Errorf("%s start: %w", t, err))
			continue
		}
		fmt.Printf("  %s: 起動しました\n", t)
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
			fmt.Printf("  %s: 停止失敗 → %v\n", t, err)
			errs = append(errs, fmt.Errorf("%s stop: %w", t, err))
			continue
		}
		fmt.Printf("  %s: 停止しました\n", t)
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
			fmt.Printf("  %s: 状態取得失敗 → %v\n", t, err)
			continue
		}

		info, discoverErr := engine.DiscoverEngine(t)
		binaryStatus := "未検出"
		if discoverErr == nil {
			binaryStatus = info.BinaryPath
		}

		installMark := "未インストール"
		if status.Installed {
			installMark = "インストール済み"
		}

		runMark := "停止"
		if status.Running {
			runMark = fmt.Sprintf("稼働中 (PID: %d)", status.PID)
		}

		fmt.Printf("  %s:\n", t)
		fmt.Printf("    バイナリ:   %s\n", binaryStatus)
		fmt.Printf("    サービス:   %s [%s]\n", installMark, status.Label)
		fmt.Printf("    状態:       %s\n", runMark)
	}
	return nil
}
