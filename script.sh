#!/bin/bash

# 定义要扫描的目录，默认为当前目录
TARGET_DIR=${1:-.}

# 查找所有.md文件并处理
find "$TARGET_DIR" -type f -name "*.md" | while read -r file; do
    # 创建临时文件
    tmpfile=$(mktemp)
    
    # 获取不带路径和扩展名的文件名
    filename=$(basename "$file" .md)
    
    # 生成标题，将-和_替换为空格，并首字母大写
    title=$(echo "$filename" | sed -E 's/[-_]/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)}1')
    
    # 生成描述
    description="Documentation for $title"
    
    # 写入新的元数据
    echo "---" > "$tmpfile"
    echo "title: $title" >> "$tmpfile"
    echo "description: $description" >> "$tmpfile"
    echo "---" >> "$tmpfile"
    echo "" >> "$tmpfile"
    
    # 移除可能的旧元数据（如果存在），并追加剩余内容
    if head -n 1 "$file" | grep -q '^---$'; then
        # 如果文件以元数据开头，找到第二个 `---` 并跳过
        awk '/^---$/ { if (++count == 2) { skip=0 } next } skip { next } { print }' "$file" >> "$tmpfile"
    else
        # 如果没有元数据，直接追加整个文件
        cat "$file" >> "$tmpfile"
    fi
    
    # 替换原文件
    mv "$tmpfile" "$file"
    
    echo "Updated metadata for: $file"
done

echo "Metadata update complete."