# -*- coding: utf-8 -*-
# 功能:
# 1. 引导框架负责自动扫描、文件I/O和多语言管理。
# 2. 核心翻译逻辑完全由 translation_client.py 模块驱动。
# 3. 动态读取原生i18n配置。
# 4. 统一处理.md .mdx 和 meta.json 文件的翻译流程。

import os
import re
import sys
import json
import hashlib
from typing import Dict, List, Tuple
from pathlib import Path

try:
    from translation_client import translate_files
except ImportError as e:
    print(f"错误：缺少必要的库 ({e})。")
    sys.exit(1)

# ==============================================================================
# 动态路径配置区
# ==============================================================================
try:
    SCRIPT_DIR = Path(__file__).resolve().parent
    PROJECT_ROOT = SCRIPT_DIR.parent.parent
except NameError:
    # 兼容在 notebook 等环境中运行
    SCRIPT_DIR = Path.cwd()
    PROJECT_ROOT = SCRIPT_DIR

# ==============================================================================
# 启动前的配置区
# ==============================================================================
SCAN_DIRECTORIES = [
    PROJECT_ROOT / 'content' / 'docs',
    PROJECT_ROOT / 'messages'
]
I18N_CONFIG_PATH = PROJECT_ROOT / 'src' / 'i18n' / 'config.ts'

# ==============================================================================
# 引导框架
# ==============================================================================

def get_i18n_config() -> Tuple[str, Dict[str, str]]:
    """从 Starlight 配置文件中读取默认语言和支持的语言列表。"""
    print(f"🤖 正在从 '{I18N_CONFIG_PATH}' 读取原生i18n配置...")
    try:
        with open(I18N_CONFIG_PATH, 'r', encoding='utf-8') as f:
            content = f.read()
    except FileNotFoundError:
        print(f"❌ 错误：配置文件 '{I18N_CONFIG_PATH}' 未找到！请检查路径。")
        sys.exit(1)
        
    default_locale_match = re.search(r"defaultLocale:.*=\s*['\"](\w+)['\"]", content)
    if not default_locale_match:
        print(f"❌ 错误：无法在 '{I18N_CONFIG_PATH}' 中找到 'defaultLocale'。")
        sys.exit(1)
    default_locale = default_locale_match.group(1)

    locales_match = re.search(r"supportedLocales\s*=\s*{(.*?)}", content, re.DOTALL)
    if not locales_match:
        print(f"❌ 错误：无法在 '{I18N_CONFIG_PATH}' 中找到 'supportedLocales'。")
        sys.exit(1)
    supported_locales_str = locales_match.group(1)
    
    locale_pairs = re.findall(r"(\w+):\s*['\"](.*?)['\"]", supported_locales_str)
    locales_map = {key.strip(): value.strip() for key, value in locale_pairs}
    
    print(f"✅ 配置读取成功: 默认语言='{default_locale}', 支持的语言={list(locales_map.keys())}")
    return default_locale, locales_map


def get_path_prefix_and_lang(file_path_str: str, default_locale: str, supported_locales: List[str]) -> Tuple[str, str]:
    """解析文件路径，返回其语言无关的前缀和语言代码。"""
    file_path = Path(file_path_str)
    dir_path = file_path.parent
    base_name = file_path.stem

    # 模式 1: 检查 'name.lang' 格式, e.g., 'index.en' from 'index.en.mdx'
    base_name_parts = base_name.split('.')
    if len(base_name_parts) > 1 and base_name_parts[-1] in supported_locales:
        lang = base_name_parts[-1]
        path_prefix = str(dir_path / ".".join(base_name_parts[:-1]))
        return path_prefix, lang

    # 模式 2: 检查 'lang' 格式, e.g., 'en' from 'en.json'
    if base_name in supported_locales:
        lang = base_name
        # 对于这种模式，文档家族的前缀就是它所在的目录
        path_prefix = str(dir_path)
        return path_prefix, lang

    # 模式 3: 默认语言文件, e.g., 'index' from 'index.mdx'
    lang = default_locale
    path_prefix = str(dir_path / base_name)
    return path_prefix, lang

def main(update_existing: bool):
    """主执行函数，扫描文件并调用翻译模块。"""
    print("\n🚀 欢迎使用i18n自动化引导程序！")
    mode = "更新现有翻译" if update_existing else "创建缺失翻译"
    print(f"当前模式: {mode}")
    
    default_locale, locales_map = get_i18n_config()
    supported_locales = list(locales_map.keys())

    # --- 步骤 1: 扫描所有指定目录，建立文档家族 ---
    doc_families: Dict[str, Dict[str, Path]] = {}
    print("\n🔍 正在扫描以下目录:")
    for directory in SCAN_DIRECTORIES:
        print(f"  - {directory.relative_to(PROJECT_ROOT)}")
        for root, _, files in os.walk(directory):
            for file in files:
                if not (file.endswith('.mdx') or file.endswith('.json') or file.endswith('.md')):
                    continue
                
                file_path = Path(root) / file
                # 假设 get_path_prefix_and_lang 能正确处理路径
                path_prefix, lang = get_path_prefix_and_lang(str(file_path), default_locale, supported_locales)
                
                if path_prefix not in doc_families:
                    doc_families[path_prefix] = {}
                doc_families[path_prefix][lang] = file_path
            
    print(f"📊 扫描完成，共找到 {len(doc_families)} 个文档家族。")

    # --- 步骤 2: 遍历每个家族，根据模式执行翻译 ---
    for prefix, files_map in doc_families.items():
        # (这部分用于打印相对路径，可以保持不变)
        relative_prefix_str = prefix.replace(str(PROJECT_ROOT), '').lstrip('/')
        print(f"\n➡️ 正在处理文档家族: '{relative_prefix_str}'")
        
        source_lang, source_file_path = "", Path()
        if default_locale in files_map:
            source_lang = default_locale
            source_file_path = files_map[default_locale]
        else:
            if not files_map: continue
            # 如果默认语言文件不存在，则选择找到的第一个作为源文件
            source_lang, source_file_path = next(iter(files_map.items()))

        print(f"  - 源文件: '{source_file_path.relative_to(PROJECT_ROOT)}'")

        # --- 核心逻辑: 根据 update_existing 参数决定行为 ---
        if update_existing:
            # --- 模式 A: 更新已有的翻译 ---
            target_langs = [lang for lang in supported_locales if lang in files_map and lang != source_lang]
            if not target_langs:
                print("  - 未找到任何已存在的其他语言版本进行更新。")
                continue
            
            print(f"  - 准备更新以下语言: {target_langs}")
            
            # 准备文件路径列表给增量翻译函数
            # 格式: [源文件, 目标文件1, 目标文件2, ...]
            paths_for_translation = [str(source_file_path)] + [str(files_map[lang]) for lang in target_langs]
            
            translated_contents = translate_files(
                file_paths=paths_for_translation,
                source_language=source_lang,
                source_language_full=locales_map[source_lang],
                target_languages=target_langs,
                target_language_full=[locales_map[lang] for lang in target_langs],
            )
            
            # 覆盖写入更新后的文件
            for lang, content in translated_contents.items():
                target_path = files_map[lang] # 路径已经存在
                with open(target_path, 'w', encoding='utf-8') as f:
                    f.write(content)
                print(f"  - ✅ 已更新翻译文件: '{target_path.relative_to(PROJECT_ROOT)}'")

        else:
            # --- 模式 B: 创建缺失的翻译 (原始逻辑) ---
            target_langs = [lang for lang in supported_locales if lang not in files_map]
            if not target_langs:
                print("  - 所有语言版本已存在，无需创建。")
                continue

            print(f"  - 准备为以下缺失语言创建翻译: {target_langs}")
            
            # 只需传入源文件路径进行完全翻译
            translated_contents = translate_files(
                file_paths=[str(source_file_path)],
                source_language=source_lang,
                source_language_full=locales_map[source_lang],
                target_languages=target_langs,
                target_language_full=[locales_map[lang] for lang in target_langs],
            )
            
            # 写入新创建的文件
            for lang, content in translated_contents.items():
                source_suffix = source_file_path.suffix
                target_path: Path

                # === 核心修正逻辑 ===
                # 检查源文件的文件名（不含后缀）本身是否就是一个支持的语言代码。
                # 这能准确识别出 'zh.json' 这类文件。
                if source_file_path.stem in supported_locales:
                    # 对于 'zh.json' 这种情况, prefix 是目录 '.../messages'
                    # 正确的路径应该是 目录 / 新语言代码.后缀
                    # 例如: '.../messages' / 'ko.json'
                    target_path = Path(prefix) / f"{lang}{source_suffix}"
                else:
                    # 对于 'index.mdx' 或 'index.zh.mdx' 这类文件，使用原始逻辑
                    # prefix 是 '.../index'
                    # 正确的路径是 前缀.新语言代码.后缀
                    # 例如: '.../index.ko.mdx'
                    if lang == default_locale:
                        # 默认语言不需要语言代码后缀
                        target_path = Path(f"{prefix}{source_suffix}")
                    else:
                        target_path = Path(f"{prefix}.{lang}{source_suffix}")
                
                # 创建可能不存在的父目录
                target_path.parent.mkdir(parents=True, exist_ok=True)
                
                with open(target_path, 'w', encoding='utf-8') as f:
                    f.write(content)
                print(f"  - ✅ 已创建翻译文件: '{target_path.relative_to(PROJECT_ROOT)}'")

    print("\n🎉🎉🎉 引导过程全部完成！🎉🎉🎉")

if __name__ == "__main__":
    UPDATE_MODE = False
    
    main(update_existing=UPDATE_MODE)