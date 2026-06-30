package engine

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
)

// xmlEscape escapes a string for safe inclusion in an XML text node, using the
// stdlib escaper. Built-in engines use only alphanumeric/path characters, so
// their output is unchanged; user-controlled values containing <, >, &, ", '
// are neutralized, preventing plist structure injection.
func xmlEscape(s string) string {
	var buf bytes.Buffer
	// xml.EscapeText only fails if the underlying writer fails; bytes.Buffer
	// never does, so the error is safe to ignore.
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// RenderPlist builds the launchd plist contents for a managed engine.
//
// For built-in engines (no Dir/Env) the output is byte-identical to the
// historical embedded templates: a fixed-key dict with Label, ProgramArguments
// (Command followed by Args), RunAtLoad, KeepAlive and the two log paths.
// User-defined engines additionally emit WorkingDirectory and
// EnvironmentVariables blocks when Dir/Env are set. All text nodes derived from
// user input are XML-escaped to prevent plist structure injection.
func RenderPlist(def *EngineDef, logDir string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"` + "\n")
	b.WriteString(`  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n")
	b.WriteString("<dict>\n")

	label := def.ServiceLabel()
	b.WriteString("    <key>Label</key>\n")
	fmt.Fprintf(&b, "    <string>%s</string>\n", xmlEscape(label))

	b.WriteString("    <key>ProgramArguments</key>\n")
	b.WriteString("    <array>\n")
	fmt.Fprintf(&b, "        <string>%s</string>\n", xmlEscape(def.Command))
	for _, a := range def.Args {
		fmt.Fprintf(&b, "        <string>%s</string>\n", xmlEscape(a))
	}
	b.WriteString("    </array>\n")

	if def.Dir != "" {
		b.WriteString("    <key>WorkingDirectory</key>\n")
		fmt.Fprintf(&b, "    <string>%s</string>\n", xmlEscape(def.Dir))
	}

	if len(def.Env) > 0 {
		b.WriteString("    <key>EnvironmentVariables</key>\n")
		b.WriteString("    <dict>\n")
		for _, k := range sortedKeys(def.Env) {
			fmt.Fprintf(&b, "        <key>%s</key>\n", xmlEscape(k))
			fmt.Fprintf(&b, "        <string>%s</string>\n", xmlEscape(def.Env[k]))
		}
		b.WriteString("    </dict>\n")
	}

	b.WriteString("    <key>RunAtLoad</key>\n")
	b.WriteString("    <true/>\n")
	b.WriteString("    <key>KeepAlive</key>\n")
	b.WriteString("    <true/>\n")
	b.WriteString("    <key>StandardOutPath</key>\n")
	fmt.Fprintf(&b, "    <string>%s/%s.stdout.log</string>\n", xmlEscape(logDir), xmlEscape(label))
	b.WriteString("    <key>StandardErrorPath</key>\n")
	fmt.Fprintf(&b, "    <string>%s/%s.stderr.log</string>\n", xmlEscape(logDir), xmlEscape(label))

	b.WriteString("</dict>\n")
	b.WriteString("</plist>\n")
	return b.String()
}

// RenderSystemdUnit builds the systemd user unit contents for a managed engine.
//
// For built-in engines (no Dir/Env) the output matches the historical embedded
// templates. User-defined engines additionally emit WorkingDirectory and
// Environment= lines. Newlines/NUL in user input are rejected upstream
// (BuildRegistry), and Environment values are quoted/escaped here, so directive
// injection into the [Service] section is not possible.
func RenderSystemdUnit(def *EngineDef) string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s Engine\n", def.DisplayName)
	b.WriteString("After=network.target\n")
	b.WriteString("\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	fmt.Fprintf(&b, "ExecStart=%s\n", systemdExecStart(def.Command, def.Args))
	if def.Dir != "" {
		fmt.Fprintf(&b, "WorkingDirectory=%s\n", def.Dir)
	}
	for _, k := range sortedKeys(def.Env) {
		fmt.Fprintf(&b, "Environment=%s\n", systemdEnvAssignment(k, def.Env[k]))
	}
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=5\n")
	b.WriteString("StandardOutput=journal\n")
	b.WriteString("StandardError=journal\n")
	b.WriteString("\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=default.target\n")
	return b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// systemdExecStart renders the ExecStart command line. Built-in engines use
// simple space-separated args (matching the historical template); arguments
// containing whitespace or systemd-special characters are double-quoted with
// the special characters escaped. Newlines are rejected upstream, so this only
// needs to handle in-line metacharacters.
func systemdExecStart(command string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, systemdArg(command))
	for _, a := range args {
		parts = append(parts, systemdArg(a))
	}
	return strings.Join(parts, " ")
}

// systemdArg quotes a single ExecStart argument when needed. systemd treats
// space, tab, quotes, backslash and the '%' specifier expander as special.
func systemdArg(a string) string {
	if a == "" {
		return `""`
	}
	if !strings.ContainsAny(a, " \t\"'\\%") {
		return a
	}
	return systemdQuote(a)
}

// systemdQuote double-quotes a value, escaping backslash, double-quote and the
// systemd '%' specifier introducer (per systemd.syntax / systemd.service).
func systemdQuote(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`%`, `%%`,
	)
	return `"` + r.Replace(s) + `"`
}

// systemdEnvAssignment renders a single Environment= assignment as a quoted
// "KEY=value" pair, which systemd parses as one whole assignment regardless of
// spaces in the value. Backslash, quote and '%' are escaped.
func systemdEnvAssignment(key, value string) string {
	return systemdQuote(key + "=" + value)
}
