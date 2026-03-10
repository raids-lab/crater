#!/bin/sh

# Check for hardcoded Crater Helm Chart versions in staged documentation files
# This script ensures that developers use components or <chart-version> placeholders
# instead of hardcoding version numbers (e.g., 0.1.0).

# Get repository root directory
GIT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
if [ -z "$GIT_ROOT" ]; then
    echo "❌ Error: Not in a git repository"
    exit 1
fi

# Change to repository root to ensure consistent paths
cd "$GIT_ROOT" || exit 1

# Get list of staged documentation files (MD/MDX) in website/
STAGED_DOC_FILES=$(git diff --cached --name-only -- "website/content/**" | grep -E "\.(md|mdx)$")

# If no staged doc files, exit successfully
if [ -z "$STAGED_DOC_FILES" ]; then
    exit 0
fi

FOUND_HARDCODED=0

# Pattern to detect hardcoded versions in Helm commands related to Crater
# Matches: ghcr.io/raids-lab/crater ... --version 0.1.0
# Matches: --version 0.1.0 ... ghcr.io/raids-lab/crater
# Uses [[:space:]] instead of \s for compatibility with BSD grep
SEMVER='[0-9]+\.[0-9]+\.[0-9]+([-][a-zA-Z0-9.]+)?([+][a-zA-Z0-9.]+)?'
PATTERN="ghcr.io/raids-lab/crater.*--version[[:space:]]+$SEMVER|--version[[:space:]]+$SEMVER.*ghcr.io/raids-lab/crater"

echo "🔍 Checking for hardcoded Crater Chart versions in staged docs..."

for file in $STAGED_DOC_FILES; do
    # Skip if file was deleted
    if [ ! -f "$file" ]; then
        continue
    fi

    # Search for the pattern in the file
    MATCHES=$(grep -nE "$PATTERN" "$file")
    
    if [ -n "$MATCHES" ]; then
        echo "❌ Error: Found hardcoded Crater Chart version in '$file':"
        echo "$MATCHES" | sed 's/^/   /'
        FOUND_HARDCODED=1
    fi
done

if [ $FOUND_HARDCODED -eq 1 ]; then
    echo ""
    echo "❌ Pre-commit check failed: Hardcoded Crater Helm Chart versions detected."
    echo "💡 Please use <CraterChartVersionNotice /> component or <chart-version> placeholder instead."
    echo "   See .github/instructions/docs.instructions.md for details."
    exit 1
fi

echo "✅ No hardcoded Crater Chart versions found in staged docs."
exit 0
