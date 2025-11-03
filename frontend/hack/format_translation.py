import os
import sys
import json

def flatten_json(y, prefix=''):
    out = {}
    if isinstance(y, dict):
        for k, v in y.items():
            new_key = f"{prefix}.{k}" if prefix else k
            out.update(flatten_json(v, new_key))
    else:
        out[prefix] = y
    return out

def process_folder(folder_path):
    processed_files = []
    
    for root, dirs, files in os.walk(folder_path):
        for file in files:
            if file == "translation.json":
                file_path = os.path.join(root, file)
                relative_path = os.path.relpath(file_path, folder_path)
                
                with open(file_path, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                    flat = flatten_json(data)
                    # Sort in ascending order
                    sorted_result = dict(sorted(flat.items()))
                
                # Write back to the original file
                with open(file_path, 'w', encoding='utf-8') as f:
                    json.dump(sorted_result, f, ensure_ascii=False, indent=2)
                
                processed_files.append(relative_path)
    
    return processed_files

if __name__ == "__main__":
    folder = "src/i18n"
    processed_files = process_folder(folder)
    
    # Log processed files
    print(f"Processed and updated {len(processed_files)} translation.json file(s):")
    for file_path in processed_files:
        print(f"  - {file_path}")