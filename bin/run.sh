#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[ -f .env ] && set -a && source .env && set +a
go run "$SCRIPT_DIR/../cmd/cache-population/main.go"