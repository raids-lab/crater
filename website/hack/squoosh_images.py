# Copyright 2025 Crater
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import re
import argparse
from PIL import Image
from pathlib import Path
from collections import defaultdict

def find_all_images(content_dir):
    """扫描所有图片文件，返回相对 content 的路径集合"""
    image_exts = {'.png', '.jpg', '.jpeg', '.gif', '.bmp', '.webp'}
    image_paths = set()
    
    for root, _, files in os.walk(content_dir):
        for file in files:
            if Path(file).suffix.lower() in image_exts:
                rel_path = os.path.relpath(os.path.join(root, file), content_dir)
                image_paths.add(rel_path.replace(os.sep, '/'))
    
    return image_paths

def extract_image_refs(content_dir):
    """提取所有 md/mdx 文件中的图片引用，返回（图片相对 content 的路径，原始引用字符串）映射"""
    refs = defaultdict(set)
    md_file_exts = {'.md', '.mdx'}
    
    for root, _, files in os.walk(content_dir):
        for file in files:
            if Path(file).suffix.lower() in md_file_exts:
                md_path = os.path.join(root, file)
                with open(md_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                
                # 查找所有图片引用
                matches = re.findall(r'!\[[^\]]*\]\(([^)]+)\)', content)
                if not matches:
                    continue
                
                # 计算当前 md 文件所在目录（相对 content）
                md_rel_dir = os.path.relpath(root, content_dir)
                
                for img_ref in matches:
                    # 处理相对路径
                    if img_ref.startswith(('./', '../')):
                        abs_ref = os.path.normpath(os.path.join(md_rel_dir, img_ref))
                        abs_ref = abs_ref.replace(os.sep, '/')
                        refs[abs_ref].add(img_ref)
                    # 处理绝对路径（从 content 开始）
                    else:
                        refs[img_ref.lstrip('/')].add(img_ref)
    
    return refs

def compress_image(img_path, max_size=1080, quality=50):
    """压缩图片并转换为 webp 格式"""
    try:
        with Image.open(img_path) as img:
            # 调整大小（保持比例）
            if max(img.size) > max_size:
                ratio = max_size / max(img.size)
                new_size = (int(img.width * ratio), int(img.height * ratio))
                img = img.resize(new_size, Image.LANCZOS)
            
            # 转换为 webp
            webp_path = f"{os.path.splitext(img_path)[0]}.webp"
            img.save(webp_path, 'WEBP', quality=quality)
            return webp_path
    except Exception as e:
        print(f"Error processing {img_path}: {str(e)}")
        return None

def main():
    parser = argparse.ArgumentParser(description='Optimize images for documentation')
    parser.add_argument('--content', default='content', help='Content directory path')
    parser.add_argument('--max-size', type=int, default=1080, help='Max image dimension')
    parser.add_argument('--quality', type=int, default=75, help='WebP compression quality (0-100)')
    parser.add_argument('--delete-unused', action='store_true', help='Delete unused images without confirmation')
    args = parser.parse_args()

    content_dir = os.path.abspath(args.content)
    print(f"Scanning content directory: {content_dir}")

    # 步骤 1: 扫描所有图片
    all_images = find_all_images(content_dir)
    print(f"Found {len(all_images)} images")

    # 步骤 2: 提取所有引用
    image_refs = extract_image_refs(content_dir)
    referenced_images = set(image_refs.keys())
    print(f"Found {len(referenced_images)} referenced images")

    # 步骤 3: 检查未使用图片
    unused_images = all_images - referenced_images
    if unused_images:
        print(f"\nFound {len(unused_images)} UNUSED images:")
        for img in sorted(unused_images):
            print(f"  - {img}")
        
        if args.delete_unused:
            do_delete = True
        else:
            response = input("\nDelete these unused images? [y/N] ").strip().lower()
            do_delete = response == 'y'
        
        if do_delete:
            for img in unused_images:
                os.remove(os.path.join(content_dir, img))
            print(f"Deleted {len(unused_images)} unused images")
    else:
        print("\nNo unused images found")

    # 步骤 4: 处理需要压缩的图片
    conversion_map = {}
    for img_rel_path, ref_strings in image_refs.items():
        img_abs_path = os.path.join(content_dir, img_rel_path)
        
        # 跳过已经是 webp 格式的
        if img_rel_path.lower().endswith('.webp'):
            continue
            
        # 压缩图片
        webp_path = compress_image(
            img_abs_path,
            max_size=args.max_size,
            quality=args.quality
        )
        
        if webp_path:
            webp_rel = os.path.relpath(webp_path, content_dir).replace(os.sep, '/')
            conversion_map[img_rel_path] = webp_rel
            print(f"Compressed: {img_rel_path} -> {webp_rel}")
            
            # 删除原始图片
            os.remove(img_abs_path)
            print(f"Deleted original: {img_rel_path}")

    # 步骤 5: 更新 Markdown 文件中的引用
    if conversion_map:
        print("\nUpdating Markdown files...")
        md_file_exts = {'.md', '.mdx'}
        updated_files = set()
        
        for root, _, files in os.walk(content_dir):
            for file in files:
                if Path(file).suffix.lower() in md_file_exts:
                    md_path = os.path.join(root, file)
                    with open(md_path, 'r', encoding='utf-8') as f:
                        content = f.read()
                    
                    updated = False
                    for orig, new in conversion_map.items():
                        # 获取该图片的所有引用方式
                        for ref_str in image_refs[orig]:
                            # 创建新引用（保持相对路径格式）
                            new_ref = ref_str.rsplit('.', 1)[0] + '.webp'
                            
                            if ref_str in content:
                                content = content.replace(ref_str, new_ref)
                                updated = True
                    
                    if updated:
                        with open(md_path, 'w', encoding='utf-8') as f:
                            f.write(content)
                        updated_files.add(os.path.relpath(md_path, content_dir))
        
        print(f"Updated {len(updated_files)} files:")
        for f in sorted(updated_files):
            print(f"  - {f}")
    
    print("\nOperation completed successfully")

if __name__ == "__main__":
    main()