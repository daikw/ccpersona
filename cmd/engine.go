package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/daikw/ccpersona/internal/cliui"
	"github.com/daikw/ccpersona/internal/engine"
	"github.com/daikw/ccpersona/internal/persona"
	"github.com/daikw/ccpersona/internal/voice"
	"github.com/urfave/cli/v3"
)

// loadEngineRegistry builds the engine registry by merging the built-in engines
// with any user-defined engines declared in the unified config file.
//
// A missing config file is expected and falls back to built-ins only. A config
// that exists but cannot be parsed is reported by the unified runtime loader
// and ignored, so built-in engines remain available.
func loadEngineRegistry() (*engine.Registry, error) {
	cfg, err := loadEngineConfig()
	if err != nil {
		return nil, err
	}

	var userEngines map[string]engine.UserEngineConfig
	if cfg != nil && len(cfg.Engines) > 0 {
		userEngines = make(map[string]engine.UserEngineConfig, len(cfg.Engines))
		for name, ec := range cfg.Engines {
			userEngines[name] = engine.UserEngineConfig{
				BaseURL: ec.BaseURL,
				Health:  ec.Health,
				Command: ec.Command,
				Args:    ec.Args,
				Dir:     ec.Dir,
				Env:     ec.Env,
			}
		}
	}
	return engine.BuildRegistry(userEngines)
}

// loadEngineConfig loads engine declarations from the unified config.
func loadEngineConfig() (*voice.ConfigFile, error) {
	cfg, err := persona.LoadConfigWithFallback()
	if err != nil {
		return nil, err
	}
	return cfg.ToVoiceConfigFile(), nil
}

// resolveTargets builds the registry and resolves the CLI target argument into
// the engine definitions to operate on.
func resolveTargets(c *cli.Command) (*engine.Registry, []*engine.EngineDef, error) {
	reg, err := loadEngineRegistry()
	if err != nil {
		return nil, nil, err
	}
	defs, err := reg.Resolve(c.Args().First())
	if err != nil {
		return nil, nil, err
	}
	return reg, defs, nil
}

func handleEngineInstall(ctx context.Context, c *cli.Command) error {
	_, defs, err := resolveTargets(c)
	if err != nil {
		return err
	}

	mgr, err := engine.NewServiceManager()
	if err != nil {
		return err
	}

	var errs []error
	for _, def := range defs {
		if !def.Managed() {
			fmt.Printf("  %s: %s\n", cliui.Label(def.Name), cliui.Warn("external (no command in config)"))
			errs = append(errs, fmt.Errorf("%s: externally managed engine cannot be installed", def.Name))
			continue
		}
		if def.Builtin() && def.Command == "" {
			// Builtin without a discovered binary.
			fmt.Printf("  %s: binary not found (install the app first)\n", cliui.Failure(def.Name))
			errs = append(errs, fmt.Errorf("%s: engine binary not found", def.Name))
			continue
		}
		if def.Builtin() {
			fmt.Printf("  %s: binary found -> %s\n", cliui.Label(def.Name), def.Command)
		}

		if err := mgr.Install(def); err != nil {
			fmt.Printf("  %s: %s -> %v\n", cliui.Label(def.Name), cliui.Failure("install failed"), err)
			errs = append(errs, fmt.Errorf("%s install: %w", def.Name, err))
			continue
		}
		fmt.Printf("  %s: %s\n", cliui.Label(def.Name), cliui.Success("service installed"))
	}
	return errors.Join(errs...)
}

func handleEngineUninstall(ctx context.Context, c *cli.Command) error {
	_, defs, err := resolveTargets(c)
	if err != nil {
		return err
	}

	mgr, err := engine.NewServiceManager()
	if err != nil {
		return err
	}

	var errs []error
	for _, def := range defs {
		if !def.Managed() {
			fmt.Printf("  %s: %s\n", cliui.Label(def.Name), cliui.Warn("external (no command in config)"))
			errs = append(errs, fmt.Errorf("%s: externally managed engine cannot be uninstalled", def.Name))
			continue
		}
		if err := mgr.Uninstall(def); err != nil {
			fmt.Printf("  %s: %s -> %v\n", cliui.Label(def.Name), cliui.Failure("uninstall failed"), err)
			errs = append(errs, fmt.Errorf("%s uninstall: %w", def.Name, err))
			continue
		}
		fmt.Printf("  %s: %s\n", cliui.Label(def.Name), cliui.Success("service uninstalled"))
	}
	return errors.Join(errs...)
}

func handleEngineStart(ctx context.Context, c *cli.Command) error {
	_, defs, err := resolveTargets(c)
	if err != nil {
		return err
	}

	mgr, err := engine.NewServiceManager()
	if err != nil {
		return err
	}

	var errs []error
	for _, def := range defs {
		if !def.Managed() {
			fmt.Printf("  %s: %s\n", cliui.Label(def.Name), cliui.Warn("external (no command in config)"))
			errs = append(errs, fmt.Errorf("%s: externally managed engine cannot be started", def.Name))
			continue
		}
		if err := mgr.Start(def); err != nil {
			fmt.Printf("  %s: %s -> %v\n", cliui.Label(def.Name), cliui.Failure("start failed"), err)
			errs = append(errs, fmt.Errorf("%s start: %w", def.Name, err))
			continue
		}
		fmt.Printf("  %s: %s\n", cliui.Label(def.Name), cliui.Success("started"))
	}
	return errors.Join(errs...)
}

func handleEngineStop(ctx context.Context, c *cli.Command) error {
	_, defs, err := resolveTargets(c)
	if err != nil {
		return err
	}

	mgr, err := engine.NewServiceManager()
	if err != nil {
		return err
	}

	var errs []error
	for _, def := range defs {
		if !def.Managed() {
			fmt.Printf("  %s: %s\n", cliui.Label(def.Name), cliui.Warn("external (no command in config)"))
			errs = append(errs, fmt.Errorf("%s: externally managed engine cannot be stopped", def.Name))
			continue
		}
		if err := mgr.Stop(def); err != nil {
			fmt.Printf("  %s: %s -> %v\n", cliui.Label(def.Name), cliui.Failure("stop failed"), err)
			errs = append(errs, fmt.Errorf("%s stop: %w", def.Name, err))
			continue
		}
		fmt.Printf("  %s: %s\n", cliui.Label(def.Name), cliui.Warn("stopped"))
	}
	return errors.Join(errs...)
}

func handleEngineStatus(ctx context.Context, c *cli.Command) error {
	reg, err := loadEngineRegistry()
	if err != nil {
		return err
	}

	// A specific engine name may be passed; otherwise show all.
	var defs []*engine.EngineDef
	if target := c.Args().First(); target != "" && target != "all" {
		def, ok := reg.Get(target)
		if !ok {
			return fmt.Errorf("unknown engine: %s (available: %s)", target, joinNames(reg.Names()))
		}
		defs = []*engine.EngineDef{def}
	} else {
		defs = reg.All()
	}

	mgr, mgrErr := engine.NewServiceManager()

	for _, def := range defs {
		// Health check
		healthMark := cliui.Warn("unreachable")
		if engine.CheckHealth(def, 2*time.Second) {
			healthMark = cliui.Success("reachable")
		}

		fmt.Printf("  %s:\n", cliui.Engine(def.Name))
		fmt.Printf("    url:     %s\n", cliui.Muted(def.HealthURL()))
		fmt.Printf("    health:  %s\n", healthMark)

		if !def.Managed() {
			fmt.Printf("    service: %s\n", cliui.Muted("external (not managed by ccpersona)"))
			continue
		}

		if def.Builtin() {
			fmt.Printf("    binary:  %s\n", cliui.Muted(def.Command))
		}

		if mgrErr != nil {
			fmt.Printf("    service: %s -> %v\n", cliui.Failure("unavailable"), mgrErr)
			continue
		}

		status, err := mgr.Status(def)
		if err != nil {
			fmt.Printf("    service: %s -> %v\n", cliui.Failure("status failed"), err)
			continue
		}

		installMark := cliui.Warn("not installed")
		if status.Installed {
			installMark = cliui.Success("installed")
		}
		fmt.Printf("    service: %s %s\n", installMark, cliui.Muted("["+status.Label+"]"))

		runMark := cliui.Warn("stopped")
		if status.Running {
			runMark = fmt.Sprintf("%s (PID: %d)", cliui.Success("running"), status.PID)
		}
		fmt.Printf("    status:  %s\n", runMark)
	}
	return nil
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
