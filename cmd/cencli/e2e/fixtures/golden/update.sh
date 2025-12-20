#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../../.." && pwd)"
BINARY="$PROJECT_ROOT/bin/censys"

if [ ! -f "$BINARY" ]; then
	echo "Error: Binary not found at $BINARY"
	echo "Please run 'make censys' first"
	exit 1
fi

cd "$SCRIPT_DIR"

echo "Updating golden fixtures..."

"$BINARY" view --help > view_help.out
"$BINARY" aggregate --help > aggregate_help.out
"$BINARY" search --help > search_help.out
"$BINARY" censeye --help > censeye_help.out
"$BINARY" history --help > history_help.out
"$BINARY" > root.out

echo "✅ All golden fixtures updated"

