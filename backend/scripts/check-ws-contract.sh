#!/usr/bin/env bash
set -euo pipefail

# Validates WebSocket contract examples against docs/ws_schema/envelope.json.

go test ./backend/server/ws -run TestEnvelopeSchemaExamples
