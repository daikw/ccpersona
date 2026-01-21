package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/daikw/ccpersona/internal/persona"
	"github.com/urfave/cli/v3"
)

func handleInit(ctx context.Context, c *cli.Command) error {
	fmt.Println("🎭 ccpersona プロジェクト初期化")
	fmt.Println("")

	reader := bufio.NewReader(os.Stdin)

	// Check existing config
	existingConfig, _ := persona.LoadConfig(".")
	if existingConfig != nil {
		fmt.Printf("⚠️  既に設定があります: %s\n", existingConfig.Name)
		fmt.Print("上書きしますか？ [y/N]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("キャンセルしました")
			return nil
		}
		fmt.Println("")
	}

	// Get persona manager
	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	// List available personas
	personas, err := manager.ListPersonas()
	if err != nil {
		return err
	}

	var selectedPersona string

	if len(personas) == 0 {
		fmt.Println("📝 利用可能なペルソナがありません")
		fmt.Println("   デフォルトのペルソナを作成します")
		fmt.Println("")
		selectedPersona = "default"
	} else {
		fmt.Println("📝 利用可能なペルソナ:")
		for i, p := range personas {
			fmt.Printf("   %d. %s\n", i+1, p)
		}
		fmt.Println("")
		fmt.Printf("ペルソナを選択してください [1-%d]: ", len(personas))

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(personas) {
			fmt.Println("無効な選択です。デフォルトを使用します。")
			selectedPersona = "default"
		} else {
			selectedPersona = personas[choice-1]
		}
		fmt.Println("")
	}

	// Select AI assistant
	fmt.Println("🤖 AI アシスタントを選択してください:")
	fmt.Println("   1. Claude Code")
	fmt.Println("   2. Cursor")
	fmt.Println("   3. 両方")
	fmt.Println("")
	fmt.Print("選択 [1-3] (デフォルト: 1): ")

	assistantInput, _ := reader.ReadString('\n')
	assistantInput = strings.TrimSpace(assistantInput)

	assistantChoice := 1
	if assistantInput != "" {
		if choice, err := strconv.Atoi(assistantInput); err == nil && choice >= 1 && choice <= 3 {
			assistantChoice = choice
		}
	}
	fmt.Println("")

	// Create persona config (common for all assistants)
	config := persona.GetDefaultConfig()
	config.Name = selectedPersona

	if err := persona.SaveConfig(".", config); err != nil {
		return err
	}

	fmt.Println("✅ 設定を作成しました:")
	fmt.Println("   - .claude/persona.json")

	// Generate assistant-specific config
	switch assistantChoice {
	case 1: // Claude Code
		showClaudeCodeHookInstructions()
	case 2: // Cursor
		if err := generateCursorHooksConfig(); err != nil {
			return err
		}
		fmt.Println("   - .cursor/hooks.json")
	case 3: // Both
		showClaudeCodeHookInstructions()
		if err := generateCursorHooksConfig(); err != nil {
			return err
		}
		fmt.Println("   - .cursor/hooks.json")
	}

	fmt.Println("")
	fmt.Printf("   ペルソナ: %s\n", selectedPersona)
	fmt.Println("")
	fmt.Println("次のステップ:")
	fmt.Println("  - 'ccpersona show' で設定を確認")
	fmt.Println("  - 'ccpersona edit <name>' でペルソナを編集")
	return nil
}

func showClaudeCodeHookInstructions() {
	fmt.Println("")
	fmt.Println("📌 Claude Code のフック設定を ~/.claude/settings.json に追加してください:")
	fmt.Println("")
	fmt.Println(`   {
     "hooks": {
       "session-start": ["ccpersona hook"],
       "Stop": [{"hooks": [{"type": "command", "command": "ccpersona voice"}]}]
     }
   }`)
	fmt.Println("")
}

func generateCursorHooksConfig() error {
	// Create .cursor directory if not exists
	cursorDir := ".cursor"
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		return fmt.Errorf("failed to create .cursor directory: %w", err)
	}

	hooksConfig := `{
  "version": 1,
  "hooks": {
    "beforeSubmitPrompt": [
      {
        "command": "ccpersona hook"
      }
    ],
    "stop": [
      {
        "command": "ccpersona voice"
      }
    ]
  }
}
`

	hooksPath := filepath.Join(cursorDir, "hooks.json")
	if err := os.WriteFile(hooksPath, []byte(hooksConfig), 0644); err != nil {
		return fmt.Errorf("failed to write hooks.json: %w", err)
	}

	return nil
}

func handleList(ctx context.Context, c *cli.Command) error {
	// Deprecated: use 'init' instead (shows list interactively)
	fmt.Fprintln(os.Stderr, "⚠️  'list' is deprecated. Use 'init' instead (shows list interactively).")

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	personas, err := manager.ListPersonas()
	if err != nil {
		return err
	}

	if len(personas) == 0 {
		fmt.Println("No personas found. Create one with 'ccpersona edit <name>'")
		return nil
	}

	fmt.Println("Available personas:")
	for _, p := range personas {
		fmt.Printf("  - %s\n", p)
	}

	return nil
}

func handleCurrent(ctx context.Context, c *cli.Command) error {
	// Deprecated: use 'show' without arguments instead
	fmt.Fprintln(os.Stderr, "⚠️  'current' is deprecated. Use 'show' without arguments instead.")
	return handleShow(ctx, c)
}

func handleSet(ctx context.Context, c *cli.Command) error {
	// Deprecated: use 'init' instead (interactive selection)
	fmt.Fprintln(os.Stderr, "⚠️  'set' is deprecated. Use 'init' instead (interactive selection).")

	personaName := c.Args().Get(0)
	if personaName == "" {
		return fmt.Errorf("persona name is required")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	if !manager.PersonaExists(personaName) {
		return fmt.Errorf("persona '%s' does not exist", personaName)
	}

	config, err := persona.LoadConfig(".")
	if err != nil {
		return err
	}

	if config == nil {
		config = persona.GetDefaultConfig()
	}

	config.Name = personaName

	if err := persona.SaveConfig(".", config); err != nil {
		return err
	}

	if err := manager.ApplyPersona(personaName); err != nil {
		return err
	}

	fmt.Printf("Set active persona to: %s\n", personaName)
	return nil
}

func handleShow(ctx context.Context, c *cli.Command) error {
	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	personaName := c.Args().Get(0)
	if personaName == "" {
		// No argument: show current persona
		current, err := manager.GetCurrentPersona()
		if err != nil {
			return err
		}
		fmt.Printf("Current persona: %s\n", current)

		// Also show the content if persona exists
		if manager.PersonaExists(current) {
			fmt.Println()
			content, err := manager.ReadPersona(current)
			if err != nil {
				return err
			}
			fmt.Println(content)
		}
		return nil
	}

	// With argument: show specified persona
	content, err := manager.ReadPersona(personaName)
	if err != nil {
		return err
	}

	fmt.Println(content)
	return nil
}

func handleCreate(ctx context.Context, c *cli.Command) error {
	// Deprecated: use 'edit' instead (creates if not exists)
	fmt.Fprintln(os.Stderr, "⚠️  'create' is deprecated. Use 'edit' instead (creates if not exists).")
	return handleEdit(ctx, c)
}

func handleEdit(ctx context.Context, c *cli.Command) error {
	personaName := c.Args().Get(0)
	if personaName == "" {
		return fmt.Errorf("persona name is required")
	}

	manager, err := persona.NewManager()
	if err != nil {
		return err
	}

	// Create persona if it doesn't exist
	if !manager.PersonaExists(personaName) {
		if err := manager.CreatePersona(personaName); err != nil {
			return err
		}
		fmt.Printf("Created new persona: %s\n", personaName)
	}

	// Get the path to the persona file
	path := manager.GetPersonaPath(personaName)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	fmt.Printf("Edited persona: %s\n", personaName)
	return nil
}

func handleConfig(ctx context.Context, c *cli.Command) error {
	var configPath string
	var config *persona.Config
	var err error

	if c.Bool("global") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir := filepath.Join(homeDir, ".claude")
		configPath = filepath.Join(configDir, "persona.json")

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		config, err = persona.LoadConfig(homeDir)
		if err != nil || config == nil {
			config = persona.GetDefaultConfig()
			if err := persona.SaveConfig(homeDir, config); err != nil {
				return fmt.Errorf("failed to create global config: %w", err)
			}
		}
	} else {
		configPath = filepath.Join(".claude", "persona.json")

		config, err = persona.LoadConfig(".")
		if err != nil {
			return err
		}
		if config == nil {
			return fmt.Errorf("no project configuration found. Run 'ccpersona init' first")
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	if c.Bool("global") {
		fmt.Println("Edited global configuration")
	} else {
		fmt.Println("Edited project configuration")
	}
	return nil
}
