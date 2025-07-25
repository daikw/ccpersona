#!/usr/bin/env bash
set -euo pipefail

# Persona Router Hook for Claude Code
# This hook is called when a Claude Code session starts
# It automatically applies the appropriate persona based on project configuration

# Get the directory where ccpersona is installed
CCPERSONA_BIN="${CCPERSONA_BIN:-ccpersona}"

# Get the project directory (current working directory)
PROJECT_DIR="${PWD}"

# Log function
log() {
    echo "[persona-router] $1" >&2
}

# Check if ccpersona is available
if ! command -v "${CCPERSONA_BIN}" &> /dev/null; then
    log "ccpersona not found in PATH"
    exit 0
fi

# Check if this is a new session (first prompt)
# We can check this by looking for a session marker file
SESSION_MARKER="${HOME}/.claude/.session_${CLAUDE_SESSION_ID:-unknown}"

if [ -f "${SESSION_MARKER}" ]; then
    # Not a new session, skip
    exit 0
fi

# Mark this session as initialized
mkdir -p "$(dirname "${SESSION_MARKER}")"
touch "${SESSION_MARKER}"

# Clean up old session markers (older than 24 hours)
find "${HOME}/.claude" -name ".session_*" -mtime +1 -delete 2>/dev/null || true

# Check if project has persona configuration
if [ -f "${PROJECT_DIR}/.claude/persona.json" ]; then
    log "Found project persona configuration"
    
    # Apply the persona using the apply command
    if "${CCPERSONA_BIN}" apply &>/dev/null; then
        # Get the persona name from the configuration
        PERSONA_NAME=$(cd "${PROJECT_DIR}" && "${CCPERSONA_BIN}" current 2>/dev/null | grep -oP 'Current persona: \K.*' || echo "unknown")
        log "Applied persona: ${PERSONA_NAME}"
        
        # Output a message for the user
        echo "🎭 人格を適用したのだ: ${PERSONA_NAME}"
    else
        log "Failed to apply persona"
    fi
else
    log "No project persona configuration found"
fi

# Continue with the original prompt
exit 0