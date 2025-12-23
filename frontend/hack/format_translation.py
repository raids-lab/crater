import os
import sys
import json
import argparse

def flatten_json(y, prefix=''):
    out = {}
    if isinstance(y, dict):
        for k, v in y.items():
            new_key = f"{prefix}.{k}" if prefix else k
            out.update(flatten_json(v, new_key))
    else:
        out[prefix] = y
    return out

def format_translation_content(data):
    """格式化翻译内容，返回格式化后的 JSON 字符串"""
    flat = flatten_json(data)
    # Sort in ascending order
    sorted_result = dict(sorted(flat.items()))
    return json.dumps(sorted_result, ensure_ascii=False, indent=2)

def process_folder(folder_path, check_only=False):
    processed_files = []
    failed_files = []
    
    for root, dirs, files in os.walk(folder_path):
        for file in files:
            if file == "translation.json":
                file_path = os.path.join(root, file)
                relative_path = os.path.relpath(file_path, folder_path)
                
                with open(file_path, 'r', encoding='utf-8') as f:
                    original_content = f.read()
                    data = json.loads(original_content)
                
                formatted_content = format_translation_content(data)
                
                if check_only:
                    # 检查模式：比较格式化后的内容
                    # 使用 strip() 去除末尾空白行的差异
                    if original_content.strip() != formatted_content.strip():
                        failed_files.append(relative_path)
                    processed_files.append(relative_path)
                else:
                    # 格式化模式：写回文件
                    with open(file_path, 'w', encoding='utf-8') as f:
                        f.write(formatted_content)
                    processed_files.append(relative_path)
    
    return processed_files, failed_files

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Format translation JSON files')
    parser.add_argument('--check', action='store_true',
                       help='Check if files are formatted correctly without modifying them')
    args = parser.parse_args()
    
    folder = "src/i18n"
    processed_files, failed_files = process_folder(folder, check_only=args.check)
    
    if args.check:
        if failed_files:
            print(f"❌ {len(failed_files)} translation file(s) are not properly formatted:")
            for file_path in failed_files:
                print(f"  - {file_path}")
            print("\nPlease run 'python3 hack/format_translation.py' to format them.")
            sys.exit(1)
        else:
            print(f"✅ All {len(processed_files)} translation file(s) are properly formatted!")
    else:
        # Log processed files
        print(f"Processed and updated {len(processed_files)} translation.json file(s):")
        for file_path in processed_files:
            print(f"  - {file_path}")