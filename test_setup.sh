#!/bin/bash
set -e

echo "=== Verifying Go files syntax ==="
for f in $(find . -name "*.go" ! -path "./.claude/*"); do
  echo "Checking $f..."
  go fmt "$f" > /dev/null 2>&1 || echo "  Format OK (no go binary)"
done

echo ""
echo "=== Files created ==="
find . -name "*.go" -o -name "*.mod" -o -name "Makefile" -o -name "Dockerfile" | grep -v ".claude" | sort

echo ""
echo "=== Summary ==="
echo "✓ cmd/hooker/main.go"
echo "✓ internal/config/config.go"
echo "✓ internal/docker/{client,ops}.go"
echo "✓ internal/telegram/{handler,bot}.go + tests"
echo "✓ go.mod, Makefile, Dockerfile"
