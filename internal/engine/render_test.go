package engine

import (
	"strings"
	"testing"
)

// builtinVoicevoxPlist is the exact output the historical embedded template
// produced for VOICEVOX, with Label/BinaryPath/Port/LogDir substituted. This
// guards against drift in the generated launchd plist for built-in engines.
const builtinVoicevoxPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.voicevox.engine</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/vv-run</string>
        <string>--host</string>
        <string>127.0.0.1</string>
        <string>--port</string>
        <string>50021</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/logs/com.voicevox.engine.stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/logs/com.voicevox.engine.stderr.log</string>
</dict>
</plist>
`

func TestRenderPlist_BuiltinMatchesLegacyTemplate(t *testing.T) {
	def := &EngineDef{
		Name:        "voicevox",
		DisplayName: "VOICEVOX",
		Command:     "/usr/local/bin/vv-run",
		Args:        []string{"--host", "127.0.0.1", "--port", "50021"},
		builtinType: VOICEVOX,
	}
	got := RenderPlist(def, "/logs")
	if got != builtinVoicevoxPlist {
		t.Errorf("RenderPlist built-in drift:\n--- got ---\n%s\n--- want ---\n%s", got, builtinVoicevoxPlist)
	}
}

const builtinVoicevoxUnit = `[Unit]
Description=VOICEVOX Engine
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/vv-run --host 127.0.0.1 --port 50021
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
`

func TestRenderSystemdUnit_BuiltinMatchesLegacyTemplate(t *testing.T) {
	def := &EngineDef{
		Name:        "voicevox",
		DisplayName: "VOICEVOX",
		Command:     "/usr/local/bin/vv-run",
		Args:        []string{"--host", "127.0.0.1", "--port", "50021"},
		builtinType: VOICEVOX,
	}
	got := RenderSystemdUnit(def)
	if got != builtinVoicevoxUnit {
		t.Errorf("RenderSystemdUnit built-in drift:\n--- got ---\n%s\n--- want ---\n%s", got, builtinVoicevoxUnit)
	}
}

const userEnginePlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.ccpersona.engine.irodori</string>
    <key>ProgramArguments</key>
    <array>
        <string>uv</string>
        <string>run</string>
        <string>irodori-tts-server</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/home/me/src/Irodori-TTS-Server</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>HF_HOME</key>
        <string>/data/hf</string>
        <key>PORT</key>
        <string>8088</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/logs/com.ccpersona.engine.irodori.stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/logs/com.ccpersona.engine.irodori.stderr.log</string>
</dict>
</plist>
`

func TestRenderPlist_UserEngine(t *testing.T) {
	def := &EngineDef{
		Name:        "irodori",
		DisplayName: "irodori",
		Command:     "uv",
		Args:        []string{"run", "irodori-tts-server"},
		Dir:         "/home/me/src/Irodori-TTS-Server",
		Env:         map[string]string{"PORT": "8088", "HF_HOME": "/data/hf"},
	}
	got := RenderPlist(def, "/logs")
	if got != userEnginePlist {
		t.Errorf("RenderPlist user engine drift:\n--- got ---\n%s\n--- want ---\n%s", got, userEnginePlist)
	}
}

const userEngineUnit = `[Unit]
Description=irodori Engine
After=network.target

[Service]
Type=simple
ExecStart=uv run irodori-tts-server
WorkingDirectory=/home/me/src/Irodori-TTS-Server
Environment="HF_HOME=/data/hf"
Environment="PORT=8088"
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
`

func TestRenderSystemdUnit_UserEngine(t *testing.T) {
	def := &EngineDef{
		Name:        "irodori",
		DisplayName: "irodori",
		Command:     "uv",
		Args:        []string{"run", "irodori-tts-server"},
		Dir:         "/home/me/src/Irodori-TTS-Server",
		Env:         map[string]string{"PORT": "8088", "HF_HOME": "/data/hf"},
	}
	got := RenderSystemdUnit(def)
	if got != userEngineUnit {
		t.Errorf("RenderSystemdUnit user engine drift:\n--- got ---\n%s\n--- want ---\n%s", got, userEngineUnit)
	}
}

func TestRenderPlist_EscapesXMLMetacharacters(t *testing.T) {
	def := &EngineDef{
		Name:        "evil",
		DisplayName: "evil",
		Command:     "uv",
		Args:        []string{`</string><key>RunAtLoad</key><false/><string>x`},
		Env:         map[string]string{"K": `a&b<c>d"e`},
	}
	got := RenderPlist(def, "/logs")

	// The injected closing tag must not appear verbatim; it must be escaped.
	if strings.Contains(got, "</string><key>RunAtLoad</key><false/>") {
		t.Errorf("plist injection not escaped:\n%s", got)
	}
	if !strings.Contains(got, "&lt;/string&gt;&lt;key&gt;RunAtLoad&lt;/key&gt;") {
		t.Errorf("expected escaped injection payload, got:\n%s", got)
	}
	// Env value special chars escaped.
	if !strings.Contains(got, "a&amp;b&lt;c&gt;d&#34;e") {
		t.Errorf("env value not escaped, got:\n%s", got)
	}
}

func TestRenderSystemdUnit_QuotesSpecialChars(t *testing.T) {
	def := &EngineDef{
		Name:        "evil",
		DisplayName: "evil",
		Command:     "uv",
		Args:        []string{"--flag", `a b`, `c"d`},
		Env:         map[string]string{"K": `v with space and "quote" and %spec`},
	}
	got := RenderSystemdUnit(def)

	// Args containing spaces/quotes must be wrapped in double quotes.
	if !strings.Contains(got, `ExecStart=uv --flag "a b" "c\"d"`) {
		t.Errorf("ExecStart not quoted/escaped, got:\n%s", got)
	}
	// Env assignment quoted as a whole, with " and % escaped.
	if !strings.Contains(got, `Environment="K=v with space and \"quote\" and %%spec"`) {
		t.Errorf("Environment not quoted/escaped, got:\n%s", got)
	}
}
