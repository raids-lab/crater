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
import argparse
from typing import Dict, List, Tuple
from pathlib import Path

try:
    from translation_client import translate_files
except ImportError as e:
    print(f"错误：无法导入 'translation_client'。请确保它与 bootstrap.py 位于同一目录或在 Python 路径中。 ({e})")
    sys.exit(1)

try:
    start_point = Path(__file__).parent
except NameError:
    start_point = Path.cwd()

try:
    REPO_ROOT = Path(os.environ['REPO_ROOT']).resolve()
except KeyError:
    print("❌ 错误: 环境变量 'REPO_ROOT' 未设置。请在 GitHub Actions a workflow 中设置它。")
    # 为了本地测试，可以提供一个 fallback
    print("ℹ️ 本地测试 Fallback: 正在尝试从当前文件位置向上查找...")
    try:
        start_point = Path(__file__).parent
    except NameError:
        start_point = Path.cwd()
    
    # 向上找到包含 .git 目录的路径作为仓库根目录
    current_path = start_point.resolve()
    while not (current_path / '.git').is_dir():
        parent_path = current_path.parent
        if parent_path == current_path:
            raise FileNotFoundError("无法找到仓库根目录 (.git 文件夹)。")
        current_path = parent_path
    REPO_ROOT = current_path
    print(f"✅ 本地测试: 找到仓库根目录: {REPO_ROOT}")

# 动态地找到项目根目录
PROJECT_ROOT = REPO_ROOT / 'crater-website'
SCAN_DIRECTORIES = [
    PROJECT_ROOT / 'content' / 'docs',
    PROJECT_ROOT / 'messages'
]
I18N_CONFIG_PATH = PROJECT_ROOT / 'src' / 'i18n' / 'config.ts'
# 定义由 GitHub Actions Workflow 创建的 diff 文件缓存目录
DIFF_CACHE_DIR = REPO_ROOT / '.diff_cache'

def is_meaningful_diff(diff_text: str) -> bool:
    """判断 diff 是否包含实质性内容变更（非空格、非格式）"""
    for line in diff_text.splitlines():
        # 只检查以 + 或 - 开头的非空行，忽略空格变更
        if line.startswith('+') or line.startswith('-'):
            stripped = line[1:].strip()
            if stripped and not re.match(r'^\s*$', stripped):
                return True
    return False

def get_i18n_config() -> Tuple[str, Dict[str, str]]:
    print(f"🤖 正在从 '{I18N_CONFIG_PATH}' 读取原生i18n配置...")
    try:
        with open(I18N_CONFIG_PATH, 'r', encoding='utf-8') as f:
            content = f.read()
    except FileNotFoundError:
        print(f"❌ 错误：配置文件 '{I18N_CONFIG_PATH}' 未找到！")
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

def main(args):
    print("\n🚀 欢迎使用i18n自动化引导程序！")
    default_locale, locales_map = get_i18n_config()
    supported_locales = list(locales_map.keys())

    # --- 步骤 1: 扫描所有指定目录，建立文档家族 ---
    doc_families: Dict[str, Dict[str, Path]] = {}
    print("\n🔍 正在扫描目录...")
    for directory in SCAN_DIRECTORIES:
        for root, _, files in os.walk(directory):
            for file in files:
                if not file.endswith(('.mdx', '.json', '.md')): continue
                file_path = Path(root) / file
                path_prefix, lang = get_path_prefix_and_lang(str(file_path), default_locale, supported_locales)
                if path_prefix not in doc_families: doc_families[path_prefix] = {}
                doc_families[path_prefix][lang] = file_path
    print(f"📊 扫描完成，共找到 {len(doc_families)} 个文档家族。")

    changed_files_list = []
    diff_content_map = {}
    if args.changed_files:
        print(f"\n🔄 检测到变更文件列表，将处理受影响的文档家族。")
        raw_paths = [p.strip() for p in args.changed_files.split(',') if p.strip()]
        
        for raw_path_str in raw_paths:
            # 这里的路径已经是相对于项目根目录的，无需再处理前缀
            absolute_path = REPO_ROOT / raw_path_str
            changed_files_list.append(absolute_path)
            
            # 读取对应的 diff 文件
            try:
                diff_file_name = raw_path_str.replace(os.sep, '_') + '.diff'
                diff_file_path = DIFF_CACHE_DIR / diff_file_name
                print(f"diff_file_path: {diff_file_path}")
                if diff_file_path.is_file():
                    diff_content = diff_file_path.read_text('utf-8')
                    if is_meaningful_diff(diff_content):
                        diff_content_map[str(absolute_path)] = diff_content
                        print(f"    - 已加载文件 '{raw_path_str}' 的 diff 内容。")
                    else:
                        print(f"    - 文件 '{raw_path_str}' 的 diff 内容无实质性变更，已忽略。")
                        changed_files_list.remove(absolute_path)

                else:
                    print(f"    - 文件 '{raw_path_str}' 是新增文件，无 diff。")
            except Exception as e:
                print(f"    - 警告：读取 diff 文件 '{diff_file_path}' 时出错: {e}")
        
        affected_families = {p: fm for p, fm in doc_families.items() if any(fp in changed_files_list for fp in fm.values())}
        doc_families = affected_families
        if not doc_families:
            print("✅ 所有变更的文件都不属于任何已知文档家族，本次无需翻译。")
            sys.exit(0)
        print(f"  - 共 {len(doc_families)} 个文档家族受到影响。")

    # --- 步骤 3: 遍历受影响的家族，智能执行翻译 ---
    for prefix, files_map in doc_families.items():
        relative_prefix_str = prefix.replace(str(PROJECT_ROOT), '').lstrip(os.sep)
        print(f"\n➡️ 正在处理文档家族: '{relative_prefix_str}'")

        source_of_truth_lang, source_of_truth_path = None, None
        family_changed_files = {lang: path for lang, path in files_map.items() if path in changed_files_list}
        
        # 确定翻译基准
        if default_locale in family_changed_files:
            # 优先规则：如果默认语言文件被修改，它就是源头
            source_of_truth_lang, source_of_truth_path = default_locale, family_changed_files[default_locale]
            print(f"  - 策略：检测到默认语言 '{default_locale}' 文件被修改，将以它为基准。")
        elif len(family_changed_files) == 1:
            # 次要规则：如果只有一个非默认语言文件被修改
            source_of_truth_lang, source_of_truth_path = list(family_changed_files.items())[0]
            print(f"  - 策略：检测到只有 '{source_of_truth_lang}' 文件被修改，将以它为基准。")
        elif len(family_changed_files) > 1:
            # 冲突规则：修改了多个非默认语言文件，意图不明
            print(f"  - ❌ 错误：检测到同家族内有多个非默认语言文件被修改 ({list(family_changed_files.keys())})。无法确定翻译基准，跳过此家族。")
            continue
        else:
            # Fallback 规则：没有文件被修改（例如，全局添加新语言），或变更的文件是新增的
            if default_locale in files_map:
                source_of_truth_lang, source_of_truth_path = default_locale, files_map[default_locale]
                print(f"  - 策略：未检测到文件变更，使用默认语言 '{default_locale}' 为基准。")
            elif files_map:
                source_of_truth_lang, source_of_truth_path = list(files_map.items())[0]
                print(f"  - 策略：未检测到文件变更且默认语言文件不存在，使用找到的第一个语言 '{source_of_truth_lang}' 为基准。")
            else:
                # 这种情况理论上不会发生，因为家族不为空
                print(f"  - ❌ 错误：文档家族为空，无法确定源文件。")
                continue
        
        print(f"  - 基准文件: '{source_of_truth_path.relative_to(PROJECT_ROOT)}'")

        # --- 3.2 识别需要创建和需要更新的目标 ---
        targets_to_create = [lang for lang in supported_locales if lang not in files_map]
        targets_to_update = [lang for lang in supported_locales if lang in files_map and lang != source_of_truth_lang]

        # --- 3.3 执行翻译 ---
        # (A) 创建缺失的语言文件
        if targets_to_create:
            print(f"  - 任务：准备为以下缺失语言创建新文件: {targets_to_create}")
            # 对于创建，我们只提供源文件，让 client 生成新内容
            creation_results = translate_files(
                file_paths=[str(source_of_truth_path)],
                source_language=source_of_truth_lang,
                source_language_full=locales_map[source_of_truth_lang],
                target_languages=targets_to_create,
                target_language_full=[locales_map.get(lang, lang) for lang in targets_to_create]
            )
            for lang, content in creation_results.items():
                source_suffix = source_of_truth_path.suffix
                if source_of_truth_path.stem in supported_locales:
                    target_path = Path(prefix) / f"{lang}{source_suffix}"
                else:
                    target_path = Path(f"{prefix}.{lang}{source_suffix}") if lang != default_locale else Path(f"{prefix}{source_suffix}")
                target_path.parent.mkdir(parents=True, exist_ok=True)
                target_path.write_text(content, encoding='utf-8')
                print(f"    - ✅ 已创建: '{target_path.relative_to(PROJECT_ROOT)}'")

        if targets_to_update:
            print(f"  - 任务：准备更新以下现有文件: {targets_to_update}")
            paths_for_update = [str(source_of_truth_path)] + [str(files_map[lang]) for lang in targets_to_update]
            translate_files(
                file_paths=paths_for_update,
                source_language=source_of_truth_lang,
                source_language_full=locales_map[source_of_truth_lang],
                target_languages=targets_to_update,
                target_language_full=[locales_map.get(lang, lang) for lang in targets_to_update],
                write_to_existing_files=True,
                diff_content_map=diff_content_map
            )
            print(f"    - ✅ 更新任务已提交给翻译客户端。")

    print("\n🎉🎉🎉 引导过程全部完成！🎉🎉🎉")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="i18n 自动化翻译引导程序 (支持 Diff)")
    parser.add_argument(
        "--changed-files", 
        type=str, 
        default="",
        help="一个用逗号分隔的、相对于项目根目录的变更文件路径列表。"
    )
    parsed_args = parser.parse_args()
    main(parsed_args)