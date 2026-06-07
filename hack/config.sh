#!/bin/bash

# Configuration file management script
# Compatible with both macOS and Linux
# Usage:
#   bash hack/config.sh link      # Create symlinks (reads config dir from stdin)
#   bash hack/config.sh status    # Show status of configuration files
#   bash hack/config.sh unlink    # Remove configuration symlinks

# set -e  # Commented out to avoid early exit in some checks

# Color definitions
RED='\033[31m'
GREEN='\033[32m'
YELLOW='\033[33m'
BLUE='\033[34m'
CYAN='\033[36m'
RESET='\033[0m'

# Detect operating system
detect_os() {
    case "$(uname -s)" in
        Darwin*)
            echo "macos"
            ;;
        Linux*)
            echo "linux"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

OS=$(detect_os)

# Get absolute path (cross-platform compatible function)
get_absolute_path() {
    local path="$1"
    
    # Expand ~
    path=$(eval echo "$path")
    
    # Convert to absolute path
    if [ "$OS" = "macos" ]; then
        # macOS: Use cd and pwd to get absolute path
        if [ -d "$path" ] || [ -f "$path" ]; then
            local dir=$(cd "$(dirname "$path")" && pwd)
            echo "$dir/$(basename "$path")"
        else
            # If path doesn't exist, try using Python (macOS usually has Python)
            if command -v python3 >/dev/null 2>&1; then
                python3 -c "import os; print(os.path.abspath('$path'))"
            else
                echo "$path"
            fi
        fi
    else
        # Linux: Use realpath (if available) or readlink -f
        if command -v realpath >/dev/null 2>&1; then
            realpath "$path" 2>/dev/null || echo "$path"
        elif command -v readlink >/dev/null 2>&1; then
            readlink -f "$path" 2>/dev/null || echo "$path"
        else
            # Fallback to cd/pwd method
            local dir=$(cd "$(dirname "$path")" && pwd)
            echo "$dir/$(basename "$path")"
        fi
    fi
}

# Get relative path (cross-platform compatible function)
get_relative_path() {
    local target="$1"
    local base="$2"
    
    # Ensure paths are absolute
    target=$(get_absolute_path "$target")
    base=$(get_absolute_path "$base")
    
    # Use Python to calculate relative path (cross-platform compatible)
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "import os; print(os.path.relpath('$target', '$base'))"
    else
        # Simple fallback: if target is under base, return relative path
        if [[ "$target" == "$base"* ]]; then
            echo "${target#$base/}"
        else
            echo "$target"
        fi
    fi
}

# Get symlink target (cross-platform compatible function)
get_link_target() {
    local link_path="$1"
    
    if [ -L "$link_path" ]; then
        # Both macOS and Linux support readlink (without -f)
        local target=$(readlink "$link_path")
        
        # If relative path, convert to absolute path
        if [[ "$target" != /* ]]; then
            local link_dir=$(dirname "$link_path")
            target="$link_dir/$target"
        fi
        
        # Normalize path (remove .. and .)
        if command -v python3 >/dev/null 2>&1; then
            python3 -c "import os; print(os.path.normpath('$target'))"
        else
            echo "$target"
        fi
    else
        echo ""
    fi
}

# Configuration file list
declare -A CONFIG_FILES=(
    ["backend/.debug.env"]="backend/.debug.env"
    ["backend/kubeconfig"]="backend/kubeconfig"
    ["backend/etc/debug-config.yaml"]="backend/etc/debug-config.yaml"
    ["frontend/.env.development"]="frontend/.env.development"
    ["storage/.env"]="storage/.env"
    ["storage/etc/config.yaml"]="storage/etc/config.yaml"
)

# Get project root directory
get_project_root() {
    # Get script directory
    local script_path="${BASH_SOURCE[0]}"
    if [ -z "$script_path" ]; then
        script_path="$0"
    fi
    
    # If relative path, convert to absolute path
    if [[ "$script_path" != /* ]]; then
        script_path="$(pwd)/$script_path"
    fi
    
    local script_dir=$(cd "$(dirname "$script_path")" 2>/dev/null && pwd)
    if [ -z "$script_dir" ]; then
        # If failed, use current working directory
        script_dir="$(pwd)/hack"
    fi
    
    local project_root=$(cd "$script_dir/.." 2>/dev/null && pwd)
    if [ -z "$project_root" ]; then
        # If failed, use current working directory
        project_root="$(pwd)"
    fi
    
    echo "$project_root"
}

PROJECT_ROOT=$(get_project_root)

# Show help information
show_help() {
    echo "Configuration file management script"
    echo ""
    echo "Usage:"
    echo "  $0 link [CONFIG_DIR]     Create symlinks for configuration files"
    echo "  $0 status                Show status of configuration files"
    echo "  $0 unlink                Remove configuration symlinks (only symlinks, not regular files)"
    echo "  $0 restore               Restore configuration files from .bak backups"
    echo ""
    echo "Examples:"
    echo "  $0 link ~/develop/crater/config"
    echo "  echo '~/develop/crater/config' | $0 link"
    echo "  make config-link CONFIG_DIR=~/develop/crater/config"
    echo "  $0 status"
    echo "  $0 unlink"
    echo "  $0 restore"
}

# Function to create symlink
create_symlink() {
    local source_file="$1"
    local target_link="$2"
    local description="$3"
    
    # Check if source file exists
    if [ ! -f "$source_file" ]; then
        echo -e "${YELLOW}⚠️  Skipping $description: source file not found at $source_file${RESET}"
        return 1
    fi
    
    # If target directory doesn't exist, create it
    local target_dir=$(dirname "$target_link")
    if [ ! -d "$target_dir" ]; then
        mkdir -p "$target_dir"
        echo -e "${BLUE}Created directory: $target_dir${RESET}"
    fi
    
    # Handle existing target file
    if [ -e "$target_link" ]; then
        if [ -L "$target_link" ]; then
            # If it's a symlink, remove it directly
            rm "$target_link"
            echo -e "${YELLOW}Removed existing symlink: $description${RESET}"
        elif [ -f "$target_link" ]; then
            # If it's a regular file, backup as .bak
            local backup_file="${target_link}.bak"
            mv "$target_link" "$backup_file"
            echo -e "${YELLOW}Backed up existing file: $description -> ${description}.bak${RESET}"
        fi
    fi
    
    # Calculate relative path
    local relative_path=$(get_relative_path "$source_file" "$(dirname "$target_link")")
    
    # Create symlink
    ln -sf "$relative_path" "$target_link"
    echo -e "${GREEN}✅ Linked $description${RESET}"
    echo "   $target_link -> $relative_path"
}

# Execute link operation
do_link() {
    # Read from command line argument first, otherwise read from stdin
    if [ -n "$2" ]; then
        CONFIG_DIR_INPUT="$2"
    else
        # Show prompt
        echo -e "${BLUE}Enter the config directory path:${RESET}"
        echo -e "${YELLOW}  Examples:${RESET}"
        echo "    ~/develop/crater/config"
        echo "    /absolute/path/to/config"
        echo "    ../relative/path/to/config"
        echo ""
        echo -n "Config directory: "
        
        # Read config directory from stdin
        read -r CONFIG_DIR_INPUT || true
    fi

    if [ -z "$CONFIG_DIR_INPUT" ]; then
        echo -e "${RED}❌ Error: No config directory provided${RESET}"
        echo "Usage:"
        echo "  $0 link [CONFIG_DIR]"
        echo "  echo '/path/to/config' | $0 link"
        echo "  make config-link CONFIG_DIR=/path/to/config"
        exit 1
    fi

    # Parse path
    CONFIG_DIR=$(get_absolute_path "$CONFIG_DIR_INPUT")

    # Validate directory exists
    if [ ! -d "$CONFIG_DIR" ]; then
        echo -e "${RED}❌ Error: Config directory does not exist: $CONFIG_DIR${RESET}"
        exit 1
    fi

    # Define subdirectories
    BACKEND_CONFIG_DIR="$CONFIG_DIR/backend"
    FRONTEND_CONFIG_DIR="$CONFIG_DIR/frontend"
    STORAGE_CONFIG_DIR="$CONFIG_DIR/storage"

    echo -e "${BLUE}Using config directory: ${CONFIG_DIR}${RESET}"
    echo ""

    # Backend configurations
    echo -e "${CYAN}Linking backend configurations...${RESET}"
    create_symlink "$BACKEND_CONFIG_DIR/.debug.env" "$PROJECT_ROOT/backend/.debug.env" "backend/.debug.env"
    create_symlink "$BACKEND_CONFIG_DIR/kubeconfig" "$PROJECT_ROOT/backend/kubeconfig" "backend/kubeconfig"
    create_symlink "$BACKEND_CONFIG_DIR/debug-config.yaml" "$PROJECT_ROOT/backend/etc/debug-config.yaml" "backend/etc/debug-config.yaml"

    # Frontend configurations
    echo ""
    echo -e "${CYAN}Linking frontend configurations...${RESET}"
    create_symlink "$FRONTEND_CONFIG_DIR/.env.development" "$PROJECT_ROOT/frontend/.env.development" "frontend/.env.development"

    # Storage configurations
    echo ""
    echo -e "${CYAN}Linking storage configurations...${RESET}"
    create_symlink "$STORAGE_CONFIG_DIR/.env" "$PROJECT_ROOT/storage/.env" "storage/.env"
    create_symlink "$STORAGE_CONFIG_DIR/config.yaml" "$PROJECT_ROOT/storage/etc/config.yaml" "storage/etc/config.yaml"

    echo ""
    echo -e "${GREEN}✅ All symlinks created successfully!${RESET}"
}

# Function to check file status
check_file_status() {
    local file_path="$1"
    local description="$2"
    local is_optional="${3:-false}"  # Third parameter, defaults to false (required file)
    
    if [ ! -e "$file_path" ]; then
        if [ "$is_optional" = "true" ]; then
            # Show warning when optional file is missing
            echo -e "  ${YELLOW}⚠️ ${RESET}${description}"
            echo -e "     ${YELLOW}Status: Missing (optional)${RESET}"
        else
            # Show error when required file is missing
            echo -e "  ${RED}❌${RESET} ${description}"
            echo -e "     ${YELLOW}Status: Missing${RESET}"
        fi
        return 1
    elif [ -L "$file_path" ]; then
        local target=$(get_link_target "$file_path")
        echo -e "  ${GREEN}🔗${RESET} ${description}"
        echo -e "     ${CYAN}Status: Symlink${RESET}"
        echo -e "     ${BLUE}Target: $target${RESET}"
        return 0
    elif [ -f "$file_path" ]; then
        echo -e "  ${YELLOW}📄${RESET} ${description}"
        echo -e "     ${YELLOW}Status: Regular file${RESET}"
        # Check if backup file exists
        if [ -f "${file_path}.bak" ]; then
            echo -e "     ${CYAN}Backup: ${file_path}.bak exists${RESET}"
        fi
        return 0
    else
        echo -e "  ${RED}❓${RESET} ${description}"
        echo -e "     ${RED}Status: Unknown type${RESET}"
        return 1
    fi
}

# Execute status operation
do_status() {
    echo -e "${CYAN}Configuration File Status${RESET}"
    echo ""

    # Backend configurations
    echo -e "${BLUE}Backend:${RESET}"
    check_file_status "$PROJECT_ROOT/backend/.debug.env" "backend/.debug.env"
    check_file_status "$PROJECT_ROOT/backend/kubeconfig" "backend/kubeconfig" "true"  # kubeconfig is optional
    check_file_status "$PROJECT_ROOT/backend/etc/debug-config.yaml" "backend/etc/debug-config.yaml"

    echo ""

    # Frontend configurations
    echo -e "${BLUE}Frontend:${RESET}"
    check_file_status "$PROJECT_ROOT/frontend/.env.development" "frontend/.env.development"

    echo ""

    # Storage configurations
    echo -e "${BLUE}Storage:${RESET}"
    check_file_status "$PROJECT_ROOT/storage/.env" "storage/.env"
    check_file_status "$PROJECT_ROOT/storage/etc/config.yaml" "storage/etc/config.yaml"

    echo ""
}

# Function to remove symlink
remove_symlink() {
    local file_path="$1"
    local description="$2"
    
    if [ ! -e "$file_path" ]; then
        echo -e "  ${YELLOW}⚠️  ${description}: File does not exist, skipping${RESET}"
        return 0
    elif [ -L "$file_path" ]; then
        rm "$file_path"
        echo -e "  ${GREEN}✅ Removed symlink: ${description}${RESET}"
        return 0
    elif [ -f "$file_path" ]; then
        echo -e "  ${RED}❌ ${description}: Regular file (not a symlink), skipping${RESET}"
        echo -e "     ${YELLOW}Hint: This is a regular file, not a symlink. Use 'rm' manually if needed.${RESET}"
        return 1
    else
        echo -e "  ${YELLOW}⚠️  ${description}: Unknown file type, skipping${RESET}"
        return 1
    fi
}

# Execute unlink operation
do_unlink() {
    set -e  # Enable error checking in do_unlink
    echo -e "${BLUE}Removing configuration symlinks...${RESET}"
    echo ""

    removed_count=0
    skipped_count=0

    # Backend configurations
    echo -e "${CYAN}Backend:${RESET}"
    if remove_symlink "$PROJECT_ROOT/backend/.debug.env" "backend/.debug.env"; then
        removed_count=$((removed_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    if remove_symlink "$PROJECT_ROOT/backend/kubeconfig" "backend/kubeconfig"; then
        removed_count=$((removed_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    if remove_symlink "$PROJECT_ROOT/backend/etc/debug-config.yaml" "backend/etc/debug-config.yaml"; then
        removed_count=$((removed_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    echo ""

    # Frontend configurations
    echo -e "${CYAN}Frontend:${RESET}"
    if remove_symlink "$PROJECT_ROOT/frontend/.env.development" "frontend/.env.development"; then
        removed_count=$((removed_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    echo ""

    # Storage configurations
    echo -e "${CYAN}Storage:${RESET}"
    if remove_symlink "$PROJECT_ROOT/storage/.env" "storage/.env"; then
        removed_count=$((removed_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    if remove_symlink "$PROJECT_ROOT/storage/etc/config.yaml" "storage/etc/config.yaml"; then
        removed_count=$((removed_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    echo ""
    if [ $removed_count -gt 0 ]; then
        echo -e "${GREEN}✅ Removed $removed_count symlink(s)${RESET}"
    fi
    if [ $skipped_count -gt 0 ]; then
        echo -e "${YELLOW}⚠️  Skipped $skipped_count file(s) (not symlinks)${RESET}"
    fi
}

# Function to restore backup file
restore_backup() {
    local file_path="$1"
    local description="$2"
    local backup_file="${file_path}.bak"
    
    # Check if backup file exists
    if [ ! -f "$backup_file" ]; then
        echo -e "  ${YELLOW}⚠️  ${description}: No backup file found, skipping${RESET}"
        return 1
    fi
    
    # Handle target file
    if [ ! -e "$file_path" ]; then
        # Target file doesn't exist, restore directly
        mv "$backup_file" "$file_path"
        echo -e "  ${GREEN}✅ Restored ${description}${RESET}"
        return 0
    elif [ -L "$file_path" ]; then
        # Target file is a symlink, remove it then restore
        rm "$file_path"
        mv "$backup_file" "$file_path"
        echo -e "  ${GREEN}✅ Restored ${description} (replaced symlink)${RESET}"
        return 0
    elif [ -f "$file_path" ]; then
        # Target file is a regular file, prompt user
        echo -e "  ${RED}❌ ${description}: Regular file already exists, skipping${RESET}"
        echo -e "     ${YELLOW}Hint: Target file exists. Remove it manually if you want to restore from backup.${RESET}"
        return 1
    else
        echo -e "  ${YELLOW}⚠️  ${description}: Unknown file type, skipping${RESET}"
        return 1
    fi
}

# Execute restore operation
do_restore() {
    echo -e "${BLUE}Restoring configuration files from backups...${RESET}"
    echo ""

    restored_count=0
    skipped_count=0

    # Backend configurations
    echo -e "${CYAN}Backend:${RESET}"
    if restore_backup "$PROJECT_ROOT/backend/.debug.env" "backend/.debug.env"; then
        restored_count=$((restored_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    if restore_backup "$PROJECT_ROOT/backend/kubeconfig" "backend/kubeconfig"; then
        restored_count=$((restored_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    if restore_backup "$PROJECT_ROOT/backend/etc/debug-config.yaml" "backend/etc/debug-config.yaml"; then
        restored_count=$((restored_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    echo ""

    # Frontend configurations
    echo -e "${CYAN}Frontend:${RESET}"
    if restore_backup "$PROJECT_ROOT/frontend/.env.development" "frontend/.env.development"; then
        restored_count=$((restored_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    echo ""

    # Storage configurations
    echo -e "${CYAN}Storage:${RESET}"
    if restore_backup "$PROJECT_ROOT/storage/.env" "storage/.env"; then
        restored_count=$((restored_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    if restore_backup "$PROJECT_ROOT/storage/etc/config.yaml" "storage/etc/config.yaml"; then
        restored_count=$((restored_count + 1))
    else
        skipped_count=$((skipped_count + 1))
    fi

    echo ""
    if [ $restored_count -gt 0 ]; then
        echo -e "${GREEN}✅ Restored $restored_count file(s)${RESET}"
    fi
    if [ $skipped_count -gt 0 ]; then
        echo -e "${YELLOW}⚠️  Skipped $skipped_count file(s)${RESET}"
    fi
}

# Main logic
case "${1:-}" in
    link)
        do_link "$@"
        ;;
    status)
        do_status
        ;;
    unlink)
        do_unlink
        ;;
    restore)
        do_restore
        ;;
    help|--help|-h)
        show_help
        ;;
    "")
        echo -e "${RED}❌ Error: No command specified${RESET}"
        echo ""
        show_help
        exit 1
        ;;
    *)
        echo -e "${RED}❌ Error: Unknown command: $1${RESET}"
        echo ""
        show_help
        exit 1
        ;;
esac
