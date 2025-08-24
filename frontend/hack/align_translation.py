#!/usr/bin/env python3
"""
🌐 Translation Alignment Checker for React i18next
Checks for missing translation keys and generates missing key files
"""

import json
import os
import re
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Set


class TranslationAligner:
    """🔧 Main class for translation alignment operations"""
    
    def __init__(self, base_dir: str = None):
        # Use current working directory as default base_dir
        if base_dir is None:
            base_dir = os.getcwd()
        self.base_dir = Path(base_dir)
        self.src_dir = self.base_dir / "src"
        self.i18n_dir = self.base_dir / "src" / "i18n" / "locales"
        self.default_lang = "zhCN"
        self.timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        
        # 📝 Pattern to match t('key') or t("key") calls with improved regex
        # This pattern ensures we only capture valid translation keys
        # Uses word boundary or specific delimiters to avoid false positives like get('key')
        self.translation_pattern = re.compile(r'(?<![a-zA-Z0-9_])t\([\'"`]([a-zA-Z][a-zA-Z0-9._-]*)[\'"`]\)')
        
    def extract_translation_keys_from_file(self, file_path: Path) -> Set[str]:
        """🔍 Extract translation keys from a single source file"""
        keys = set()
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                matches = self.translation_pattern.findall(content)
                # Filter out invalid keys
                valid_keys = []
                for key in matches:
                    # Skip empty keys, very short keys, or keys that look like invalid patterns
                    if (len(key) > 1 and 
                        not key.startswith('.') and 
                        not key.endswith('.') and
                        not all(c in '.,/:\\=?#' for c in key) and
                        key not in ['a', 'tab', 'token', '/', ':', '=', '?', ',', '\n']):
                        valid_keys.append(key)
                keys.update(valid_keys)
        except Exception as e:
            print(f"⚠️  Error reading {file_path}: {e}")
        return keys
    
    def scan_source_files(self) -> Set[str]:
        """📁 Scan all source files for translation keys"""
        print("🔍 Scanning source files for translation keys...")
        all_keys = set()
        
        # File extensions to scan
        extensions = {'.tsx', '.jsx', '.ts', '.js'}
        
        for file_path in self.src_dir.rglob('*'):
            if file_path.suffix in extensions and file_path.is_file():
                keys = self.extract_translation_keys_from_file(file_path)
                if keys:
                    print(f"   📄 Found {len(keys)} keys in {file_path.relative_to(self.base_dir)}")
                all_keys.update(keys)
        
        print(f"✅ Total unique translation keys found: {len(all_keys)}")
        return all_keys
    
    def load_translation_file(self, lang: str) -> Dict:
        """📖 Load translation JSON file for a specific language"""
        file_path = self.i18n_dir / lang / "translation.json"
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                return json.load(f)
        except FileNotFoundError:
            print(f"⚠️  Translation file not found: {file_path}")
            return {}
        except json.JSONDecodeError as e:
            print(f"❌ Invalid JSON in {file_path}: {e}")
            return {}
    
    def get_nested_value(self, data: Dict, key: str):
        """🔗 Get value from dictionary (supports both flat and nested structure)"""
        # First try to get the key directly (flat structure)
        if key in data:
            return data[key]
        
        # If not found, try nested structure with dot notation
        keys = key.split('.')
        current = data
        for k in keys:
            if isinstance(current, dict) and k in current:
                current = current[k]
            else:
                return None
        return current
    
    def get_all_keys_from_dict(self, data: Dict, prefix: str = "") -> Set[str]:
        """🗝️  Extract all possible keys from dictionary (supports both flat and nested structure)"""
        keys = set()
        for key, value in data.items():
            if prefix:
                # This is for nested structure processing
                current_key = f"{prefix}.{key}"
                keys.add(current_key)
                if isinstance(value, dict):
                    keys.update(self.get_all_keys_from_dict(value, current_key))
            else:
                # First level - add the key as-is (supports flat structure like "accountDetail.addUser")
                keys.add(key)
                # Also check if it's a nested object and add nested keys
                if isinstance(value, dict):
                    keys.update(self.get_all_keys_from_dict(value, key))
        return keys
    
    def check_missing_keys_in_default(self, source_keys: Set[str]) -> List[str]:
        """❓ Check for missing keys in default language (zhCN)"""
        print(f"\n🔍 Checking for missing keys in default language ({self.default_lang})...")
        
        default_translations = self.load_translation_file(self.default_lang)
        missing_keys = []
        
        for key in source_keys:
            if self.get_nested_value(default_translations, key) is None:
                missing_keys.append(key)
        
        if missing_keys:
            print(f"❌ Found {len(missing_keys)} missing keys in {self.default_lang}:")
            for key in sorted(missing_keys):
                print(f"   🔑 {key}")
        else:
            print(f"✅ All source keys exist in {self.default_lang}")
        
        return missing_keys
     
    def create_nested_dict_from_keys(self, keys: List[str], default_translations: Dict) -> Dict:
        """🏗️  Create dictionary structure from flat keys (preserves flat structure for dot-separated keys)"""
        result = {}
        
        for key in keys:
            # Try to get the value from default translations
            default_value = self.get_nested_value(default_translations, key)
            
            # For flat structure (like "accountDetail.addUser"), just add the key directly
            if key in default_translations or '.' not in key:
                result[key] = default_value if default_value is not None else f"TODO: Translate '{key}'"
            else:
                # For truly nested structure, create nested objects
                keys_parts = key.split('.')
                current = result
                
                # Navigate/create the nested structure
                for i, part in enumerate(keys_parts[:-1]):
                    if part not in current:
                        current[part] = {}
                    current = current[part]
                
                # Set the final value
                final_key = keys_parts[-1]
                current[final_key] = default_value if default_value is not None else f"TODO: Translate '{key}'"
        
        return result
    
    def save_missing_keys_files(self, missing_by_lang: Dict[str, List[str]]):
        """💾 Save missing keys to separate JSON files"""
        if not missing_by_lang:
            print("\n✅ No missing keys to save!")
            return
        
        print(f"\n💾 Saving missing keys files...")
        default_translations = self.load_translation_file(self.default_lang)
        
        # Create output directory
        output_dir = self.base_dir / "hack" / "missing_translations"
        output_dir.mkdir(exist_ok=True)
        
        for lang, missing_keys in missing_by_lang.items():
            filename = f"{lang}_missing_{self.timestamp}.json"
            file_path = output_dir / filename
            
            # Create nested structure for missing keys
            missing_dict = self.create_nested_dict_from_keys(missing_keys, default_translations)
            
            with open(file_path, 'w', encoding='utf-8') as f:
                json.dump(missing_dict, f, ensure_ascii=False, indent=2)
            
            print(f"   📁 {lang}: {file_path}")
        
        print(f"✅ Missing translation files saved to {output_dir}")
    
    def run(self):
        """🚀 Main execution method"""
        print("🌐 Translation Alignment Checker Starting...")
        print(f"📁 Base directory: {self.base_dir}")
        print(f"🗣️  Default language: {self.default_lang}")
        print("-" * 60)
        
        # Step 1: Extract translation keys from source code
        source_keys = self.scan_source_files()
        
        # Step 2: Check missing keys in default language
        missing_in_default = self.check_missing_keys_in_default(source_keys)
        
        print("\n" + "=" * 60)
        print("📊 SUMMARY")
        print("=" * 60)
        print(f"🔑 Total source keys found: {len(source_keys)}")
        print(f"❌ Missing in {self.default_lang}: {len(missing_in_default)}")
        
        if missing_in_default or missing_by_lang:
            print("\n⚠️  Action required: Please review and add missing translations!")
        else:
            print("\n🎉 All translations are aligned!")


def main():
    """🎯 Entry point"""
    try:
        aligner = TranslationAligner()
        aligner.run()
    except KeyboardInterrupt:
        print("\n\n⏹️  Operation cancelled by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Unexpected error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
