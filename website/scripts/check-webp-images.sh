#!/bin/sh

# Check for non-WebP images in newly added files
# This script can be run from either the repository root or the website directory

# Get repository root directory
GIT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
if [ -z "$GIT_ROOT" ]; then
    echo "❌ Error: Not in a git repository"
    exit 1
fi

# Change to repository root to ensure consistent paths
cd "$GIT_ROOT" || exit 1

# Define website content directory (relative to repository root)
WEBSITE_CONTENT_DIR="website"

# Get all newly added files in the website directory
# --diff-filter=A: only show newly added files
# --name-only: only show file names
NEWLY_ADDED_FILES=$(git diff --cached --name-only --diff-filter=A -- "$WEBSITE_CONTENT_DIR" 2>/dev/null)

# If no newly added files, exit successfully
if [ -z "$NEWLY_ADDED_FILES" ]; then
    echo "No newly added files in '$WEBSITE_CONTENT_DIR' directory, skipping check."
    exit 0
fi

# Define non-WebP image extensions
NON_WEBP_EXTENSIONS=".png .jpg .jpeg .gif .bmp"

# Iterate through all newly added files to check if they are non-WebP images
FOUND_NON_WEBP=0
for file in $NEWLY_ADDED_FILES; do
    # Extract file extension (convert to lowercase)
    extension=$(echo "$file" | awk -F'.' '{print tolower($NF)}')
    
    # Check if extension is in non-WebP list
    if echo "$NON_WEBP_EXTENSIONS" | grep -q "\.${extension}"; then
        echo "❌ Error: Found newly added non-WebP image in '$file'."
        echo "   Please convert this image to WebP format before committing."
        FOUND_NON_WEBP=1
    fi
done

if [ $FOUND_NON_WEBP -eq 1 ]; then
    echo ""
    echo "💡 Please convert all images to WebP by format in '$WEBSITE_CONTENT_DIR/hack' before committing."
    exit 1
fi

echo "✅ No non-WebP images found in newly added files."
exit 0

