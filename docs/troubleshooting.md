# Troubleshooting Guide

This guide helps you diagnose and fix common issues with ccpersona.

## Quick Diagnostics

Run the built-in diagnostic tool:

```bash
ccpersona doctor
```

This checks:
- ccpersona installation and version
- Persona configuration
- Voice engine availability
- Hook configuration

## Voice Synthesis Issues

### No sound is produced

**Symptoms:**
- Command completes without error but no audio plays
- "Audio saved to..." message appears but no playback

**Solutions:**

1. **Check voice engine is running:**
   ```bash
   # For VOICEVOX
   curl -s http://localhost:50021/version

   # For AivisSpeech
   curl -s http://localhost:10101/version
   ```

2. **Check audio player is available:**
   ```bash
   # macOS
   which afplay

   # Linux
   which aplay || which paplay || which ffplay
   ```

3. **Test voice synthesis directly:**
   ```bash
   echo "テスト" | ccpersona voice --plain
   ```

4. **Check speaker ID is valid:**
   ```bash
   ccpersona voice --list-voices --provider aivisspeech
   ```

### Voice engine connection refused

**Symptoms:**
- Error: "connection refused" or "failed to synthesize"
- Works on one machine but not another

**Solutions:**

1. **Check port binding:**
   ```bash
   lsof -i :10101  # AivisSpeech
   lsof -i :50021  # VOICEVOX
   ```

2. **For remote access via SSH, use port forwarding:**
   ```bash
   ssh -R 10101:localhost:10101 user@remote-machine
   ```

3. **Check firewall settings:**
   ```bash
   # macOS
   sudo pfctl -s all | grep 10101

   # Linux
   sudo iptables -L -n | grep 10101
   ```

### Wrong voice is used

**Symptoms:**
- Voice plays but with wrong speaker

**Solutions:**

1. **Check configuration priority:**
   - CLI flags override config file
   - Project config overrides global config
   - Persona voice settings override defaults

2. **Verify current configuration:**
   ```bash
   ccpersona voice config show
   ```

3. **Check speaker IDs:**
   ```bash
   ccpersona voice --list-voices --provider aivisspeech
   ```

## Persona Issues

### Persona is not applied

**Symptoms:**
- Claude Code doesn't show persona behavior
- Hook appears to run but no effect

**Solutions:**

1. **Check project configuration exists:**
   ```bash
   cat .claude/persona.json
   ```

2. **Check persona file exists:**
   ```bash
   ls ~/.claude/personas/
   ccpersona show <persona-name>
   ```

3. **Check Claude Code hooks configuration:**
   ```bash
   cat ~/.claude/settings.json | grep -A5 "session-start"
   ```

4. **Verify hook output manually:**
   ```bash
   echo '{"session_id":"test"}' | ccpersona hook
   ```

5. **Check session tracking (persona only applies once per session):**
   ```bash
   ls /tmp/ccpersona-sessions/
   # Remove to force reapplication
   rm -rf /tmp/ccpersona-sessions/
   ```

### Persona changes don't take effect

**Symptoms:**
- Edited persona but behavior unchanged

**Solutions:**

1. **Start a new Claude Code session** - personas are applied once per session

2. **Clear session tracking:**
   ```bash
   rm -rf /tmp/ccpersona-sessions/
   ```

3. **Verify the persona was saved:**
   ```bash
   ccpersona show <persona-name>
   ```

## SSH/Remote Connection Issues

### Voice synthesis fails over SSH

**Symptoms:**
- Works locally but not via SSH
- Error about audio playback

**Solutions:**

1. **Use remote port forwarding:**
   ```bash
   ssh -R 10101:localhost:10101 user@remote
   ```

2. **For persistent connection, add to SSH config:**
   ```
   # ~/.ssh/config
   Host remote-server
     HostName example.com
     User youruser
     RemoteForward 10101 localhost:10101
   ```

3. **Verify port is forwarded:**
   ```bash
   # On remote machine
   curl -s http://localhost:10101/version
   ```

### Different speaker IDs per device

To use different speakers on different machines:

1. **Create device-specific voice config:**
   ```bash
   # On each device
   mkdir -p .claude
   cat > .claude/voice.json << 'EOF'
   {
     "default_provider": "aivisspeech",
     "providers": {
       "aivisspeech": {
         "speaker": 888753760
       }
     }
   }
   EOF
   ```

2. **Or use global config per device:**
   ```bash
   mkdir -p ~/.claude
   # Edit ~/.claude/voice.json with device-specific speaker ID
   ```

## Hook Issues

### Hook not triggered

**Symptoms:**
- ccpersona commands not running automatically

**Solutions:**

1. **Verify Claude Code hooks configuration:**
   ```json
   {
     "hooks": {
       "session-start": ["ccpersona hook"],
       "Stop": [
         {
           "hooks": [
             {
               "type": "command",
               "command": "ccpersona voice"
             }
           ]
         }
       ]
     }
   }
   ```

2. **Check ccpersona is in PATH:**
   ```bash
   which ccpersona
   ```

3. **Test hook manually:**
   ```bash
   echo '{"hook_event_name":"Stop","session_id":"test","transcript_path":"/path"}' | ccpersona voice
   ```

### Hook causes Claude Code to hang

**Symptoms:**
- Claude Code becomes unresponsive after session start

**Solutions:**

1. **Check for infinite loops in custom scripts**

2. **Add timeout to hook configuration** (if supported)

3. **Test hook in isolation:**
   ```bash
   time echo '{"session_id":"test"}' | ccpersona hook
   ```

## Getting Help

If issues persist:

1. **Enable verbose logging:**
   ```bash
   ccpersona --verbose voice --plain <<< "test"
   ```

2. **Check ccpersona version:**
   ```bash
   ccpersona --version
   ```

3. **Report issues on GitHub:**
   https://github.com/daikw/ccpersona/issues
