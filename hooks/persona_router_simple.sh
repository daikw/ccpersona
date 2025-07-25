#!/usr/bin/env bash
set -euo pipefail

# Simplified Persona Router Hook for Claude Code
# This hook delegates all logic to the ccpersona binary

# Get the directory where ccpersona is installed
CCPERSONA_BIN="${CCPERSONA_BIN:-ccpersona}"

# Check if ccpersona is available
if ! command -v "${CCPERSONA_BIN}" &> /dev/null; then
    echo "[persona-router] ccpersona not found in PATH" >&2
    exit 0
fi

# Delegate to ccpersona hook command
exec "${CCPERSONA_BIN}" hook