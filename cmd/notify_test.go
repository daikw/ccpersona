package main

import (
	"strings"
	"testing"
)

func TestOsascriptNotifyArgs(t *testing.T) {
	// A message containing AppleScript metacharacters must be passed as argv data,
	// never spliced into the script source.
	msg := `"; do shell script "touch /tmp/pwned`
	args := osascriptNotifyArgs(msg, "Claude Code")

	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d: %v", len(args), args)
	}
	if args[0] != "-e" {
		t.Errorf("expected first arg -e, got %q", args[0])
	}
	if !strings.Contains(args[1], "on run argv") || !strings.Contains(args[1], "item 1 of argv") {
		t.Errorf("script should reference argv, got %q", args[1])
	}
	if strings.Contains(args[1], msg) {
		t.Errorf("message must not be embedded in the script source")
	}
	if args[2] != "--" {
		t.Errorf("expected '--' terminator before argv values, got %q", args[2])
	}
	if args[3] != msg {
		t.Errorf("message must be passed verbatim as argv, got %q", args[3])
	}
	if args[4] != "Claude Code" {
		t.Errorf("title must be passed as argv, got %q", args[4])
	}
}

func TestOsascriptNotifyArgsMessageStartingWithDashE(t *testing.T) {
	// Without the '--' terminator, osascript would treat this message as an
	// additional -e statement instead of argv data.
	msg := `-e tell application "Finder" to quit`
	args := osascriptNotifyArgs(msg, "Claude Code")

	sepIdx := -1
	for i, a := range args {
		if a == "--" {
			sepIdx = i
			break
		}
	}
	if sepIdx == -1 {
		t.Fatalf("expected '--' terminator, got %v", args)
	}
	tail := args[sepIdx+1:]
	if len(tail) != 2 || tail[0] != msg || tail[1] != "Claude Code" {
		t.Errorf("message/title must follow '--' as argv data, got %v", tail)
	}
	// The only -e before '--' must be the script statement flag itself.
	for _, a := range args[1:sepIdx] {
		if a == "-e" {
			t.Errorf("message leaked as an option before '--': %v", args)
		}
	}
}

func TestNormalizeUrgency(t *testing.T) {
	cases := map[string]string{
		"low":      "low",
		"normal":   "normal",
		"critical": "critical",
		"high":     "critical",
		"urgent":   "normal",
		"":         "normal",
	}
	for in, want := range cases {
		if got := normalizeUrgency(in); got != want {
			t.Errorf("normalizeUrgency(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNotifySendArgsNormalizesUrgency(t *testing.T) {
	args := notifySendArgs("error occurred", "high", "Claude Code")
	if args[0] != "-u" || args[1] != "critical" {
		t.Errorf("urgency 'high' must be normalized to 'critical', got %v", args[:2])
	}
}

func TestNotifySendArgs(t *testing.T) {
	// A message starting with '-' must not be parsed as an option.
	msg := "--malicious-flag injected"
	args := notifySendArgs(msg, "critical", "Claude Code")

	sepIdx := -1
	for i, a := range args {
		if a == "--" {
			sepIdx = i
			break
		}
	}
	if sepIdx == -1 {
		t.Fatalf("expected '--' separator, got %v", args)
	}
	// Everything after '--' is positional; the message must live there.
	tail := args[sepIdx+1:]
	if len(tail) != 2 || tail[0] != "Claude Code" || tail[1] != msg {
		t.Errorf("title/message must follow '--' as positionals, got %v", tail)
	}
}

func TestBuildNotificationCommandUnsupported(t *testing.T) {
	if _, err := buildNotificationCommand("plan9", "hi", "normal"); err == nil {
		t.Error("expected error for unsupported platform")
	}
}

func TestBuildNotificationCommandWindowsUsesEnv(t *testing.T) {
	cmd, err := buildNotificationCommand("windows", "hello", "normal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var foundMsg, foundTitle bool
	for _, e := range cmd.Env {
		if e == "CCPERSONA_NOTIFY_MESSAGE=hello" {
			foundMsg = true
		}
		if e == "CCPERSONA_NOTIFY_TITLE=Claude Code" {
			foundTitle = true
		}
	}
	if !foundMsg || !foundTitle {
		t.Errorf("message/title must be passed via env, env=%v", cmd.Env)
	}
	for _, a := range cmd.Args {
		if strings.Contains(a, "hello") {
			t.Errorf("message must not appear in command args: %q", a)
		}
	}
}
