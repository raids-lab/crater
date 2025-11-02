# hack/refactor_filenames.py
import os
import re
import argparse
from typing import List, Tuple, Dict, Set

def to_kebab_case(name: str) -> str:
    """Converts a string from PascalCase or camelCase to kebab-case."""
    # Special case for README files
    if name.upper() == 'README':
        return name
        
    # Don't convert names that are already kebab-case or special names
    if (name.islower() and '-' in name) or not any(c.isalpha() for c in name):
        return name
        
    # Insert a hyphen before any uppercase letter, except at the start of the string
    name = re.sub(r'(?<!^)(?=[A-Z])', '-', name)
    return name.lower()

def find_rename_targets(root_dir: str, ignore_list: List[str]) -> List[Tuple[str, str]]:
    """
    Walks the directory from the bottom up to find files/dirs to rename.
    Bottom-up is crucial to ensure directories are renamed after their contents.
    """
    rename_map = []
    abs_root = os.path.abspath(root_dir)
    
    # Normalize ignore list to be relative paths
    normalized_ignore = [os.path.normpath(p) for p in ignore_list]

    for dirpath, dirnames, filenames in os.walk(root_dir, topdown=False):
        # Process files first
        for filename in filenames:
            rel_path = os.path.relpath(os.path.join(dirpath, filename), root_dir)
            if any(rel_path.startswith(p) for p in normalized_ignore):
                continue
                
            if any(c.isupper() for c in filename):
                old_path = os.path.join(dirpath, filename)
                base, ext = os.path.splitext(filename)
                new_filename = to_kebab_case(base) + ext
                new_path = os.path.join(dirpath, new_filename)
                if old_path != new_path:
                    rename_map.append((old_path, new_path))
        
        # Then process directories
        for dirname in dirnames:
            rel_path = os.path.relpath(os.path.join(dirpath, dirname), root_dir)
            if any(rel_path.startswith(p) for p in normalized_ignore):
                continue

            if any(c.isupper() for c in dirname):
                old_path = os.path.join(dirpath, dirname)
                new_dirname = to_kebab_case(dirname)
                new_path = os.path.join(dirpath, new_dirname)
                if old_path != new_path:
                    rename_map.append((old_path, new_path))

    return rename_map

def update_file_imports(root_dir: str, rename_map: List[Tuple[str, str]], dry_run: bool):
    """
    Scans all .ts and .tsx files and updates their import statements based on the rename map.
    """
    # Create a lookup dictionary for faster checks
    # We strip extensions because imports often omit them
    old_to_new_lookup = {
        os.path.splitext(os.path.abspath(old))[0]: os.path.splitext(os.path.abspath(new))[0]
        for old, new in rename_map
    }

    if not old_to_new_lookup:
        return

    print("\n🔍 Scanning for imports to update...")
    
    # Regex to find static and dynamic imports, handles single and double quotes
    import_regex = re.compile(r"""from\s+(['"])([^'"]+)\1|import\((['"])([^'"]+)\3\)""")

    files_to_update = 0
    for dirpath, _, filenames in os.walk(root_dir):
        for filename in filenames:
            if not (filename.endswith('.tsx') or filename.endswith('.ts')):
                continue

            file_path = os.path.join(dirpath, filename)
            
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    content = f.read()
            except UnicodeDecodeError:
                print(f"⚠️  Could not read {file_path} (skipping).")
                continue

            original_content = content
            
            # Use finditer to get all matches
            matches = import_regex.finditer(content)
            
            for match in matches:
                # Group 2 is for `from '...'` and group 4 is for `import('...')`
                import_path = match.group(2) or match.group(4)
                
                if not import_path:
                    continue

                abs_path_base = None
                # Correctly resolve path based on its type
                if import_path.startswith('@/'):
                    # It's an alias path, resolve from root_dir
                    abs_path_base = os.path.abspath(os.path.join(root_dir, import_path[2:]))
                elif import_path.startswith('.'):
                    # It's a relative path, resolve from the current file's directory
                    abs_path_base = os.path.abspath(os.path.join(os.path.dirname(file_path), import_path))
                else:
                    # It's a library import, skip it
                    continue

                if abs_path_base in old_to_new_lookup:
                    new_abs_path = old_to_new_lookup[abs_path_base]
                    new_import_base_name = os.path.basename(new_abs_path)
                    
                    # Construct the new import path by replacing the last part (the filename)
                    path_parts = import_path.split('/')
                    path_parts[-1] = new_import_base_name
                    new_import_path = '/'.join(path_parts)
                    
                    # Extract the original quote style to preserve it
                    quote_char = match.group(1) or match.group(3)
                    
                    original_import_statement = f"{quote_char}{import_path}{quote_char}"
                    new_import_statement = f"{quote_char}{new_import_path}{quote_char}"
                    
                    if original_import_statement != new_import_statement:
                        print(f"  - In '{os.path.relpath(file_path, root_dir)}':")
                        print(f"    {import_path}  ->  {new_import_path}")
                        content = content.replace(original_import_statement, new_import_statement)

            if content != original_content:
                files_to_update += 1
                if not dry_run:
                    print(f"  💾 Writing changes to '{os.path.relpath(file_path, root_dir)}'")
                    with open(file_path, 'w', encoding='utf-8') as f:
                        f.write(content)
    
    if files_to_update > 0:
        print(f"\n✅ Found and updated imports in {files_to_update} files.")
    else:
        print("\n✅ No relevant local imports needed updating.")


def main():
    parser = argparse.ArgumentParser(description="Refactor filenames to kebab-case and update imports.")
    parser.add_argument(
        "--dir", 
        required=True, 
        help="The root directory to scan (e.g., 'src')."
    )
    parser.add_argument(
        "--ignore", 
        action="append",
        default=[],
        help="Relative path to ignore (file or folder). Can be used multiple times."
    )
    parser.add_argument(
        "--dry-run", 
        action="store_true", 
        help="Show what would be changed without actually modifying any files."
    )
    args = parser.parse_args()

    if not os.path.isdir(args.dir):
        print(f"Error: Directory '{args.dir}' not found.")
        return

    print("="*50)
    if args.dry_run:
        print("DRY RUN MODE: No files will be changed.")
    else:
        print("⚠️  LIVE MODE: Files WILL be modified.")
    print("="*50)
    
    if args.ignore:
        print("Ignoring the following paths:")
        for p in args.ignore:
            print(f"  - {p}")

    # 1. Find all files and directories that need to be renamed
    rename_map = find_rename_targets(args.dir, args.ignore)

    if not rename_map:
        print("\n✅ All filenames already seem to be in kebab-case or are ignored. No renames needed.")
    else:
        print("\n🔄 The following renames will be performed:")
        # Sort for readability
        for old, new in sorted(rename_map):
            old_rel = os.path.relpath(old, args.dir)
            new_rel = os.path.relpath(new, args.dir)
            print(f"  - '{old_rel}'  ->  '{new_rel}'")

    # 2. Update all import statements
    update_file_imports(args.dir, rename_map, args.dry_run)

    # 3. Perform the actual renaming if not in dry-run
    if not args.dry_run and rename_map:
        print("\n🚀 Performing renames...")
        try:
            for old, new in rename_map:
                os.rename(old, new)
            print("✅ All renames completed successfully!")
        except OSError as e:
            print(f"\n❌ An error occurred during renaming: {e}")
            print("Please check the state of your files. You may need to revert using Git.")

    print("\n✨ Refactoring process finished.")
    if args.dry_run:
        print("Run without the --dry-run flag to apply the changes.")

if __name__ == "__main__":
    main()