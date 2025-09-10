import os
import requests
import json
import time
from collections import OrderedDict
import re

MAX_RETRIES = 3

# === 辅助函数: 展平 & 重建 JSON ===

def _flatten_json(nested_json: dict, separator: str = '.') -> dict:
    """
    将嵌套的字典递归地展平。
    例如: {"a": {"b": "c"}} -> {"a.b": "c"}
    非字典的值（包括列表）被视为叶节点。
    """
    flat_data = {}
    def flatten(obj, prefix=''):
        if isinstance(obj, dict):
            for key, value in obj.items():
                new_key = f"{prefix}{separator}{key}" if prefix else key
                flatten(value, new_key)
        else:
            # 如果值不是字典（字符串、数字、列表、布尔值等），则直接赋值
            flat_data[prefix] = obj
    flatten(nested_json)
    return flat_data

def _unflatten_json(flat_json: dict, separator: str = '.') -> dict:
    """
    将展平的字典重建为嵌套结构。
    例如: {"a.b": "c"} -> {"a": {"b": "c"}}
    """
    nested_data = {}
    for key, value in flat_json.items():
        parts = key.split(separator)
        d = nested_data
        for part in parts[:-1]:
            # 如果路径不存在，则创建字典
            if part not in d:
                d[part] = {}
            d = d[part]
        d[parts[-1]] = value
    return nested_data

def _call_llm_api(system_prompt: str, user_prompt: str, api_url: str, model_path: str) -> str | None:
    """
    一个独立的函数，用于调用 vLLM API 并处理响应和错误。
    返回翻译后的文本，如果失败则返回 None。
    """
    payload = {
        "model": model_path,
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_prompt}
        ],
        "max_tokens": 8192,
        "temperature": 0.7
    }
    try:
        response = requests.post(api_url, json=payload, timeout=720)
        response.raise_for_status()
        
        response_data = response.json()
        raw_content = response_data['choices'][0]['message']['content']

        # 清理可能存在的 <think> 标签
        think_tag_end = "</think>"
        think_end_index = raw_content.find(think_tag_end)
        if think_end_index != -1:
            return raw_content[think_end_index + len(think_tag_end):].strip()
        else:
            return raw_content.strip()
            
    except requests.exceptions.ConnectionError:
        print(f"[-] API 连接错误: 无法连接到 vLLM 服务。")
        # 抛出异常以便上层可以决定是否重试或终止
        raise
    except requests.exceptions.HTTPError as e:
        print(f"[-] API 请求失败，HTTP 状态码: {e.response.status_code}, 详情: {e.response.text}")
    except (KeyError, IndexError) as e:
        print(f"[-] 解析 API 响应失败: {e}")
    except Exception as e:
        print(f"[-] 调用 API 时发生未知错误: {e}")
        
    return None

# === 主函数: translate_files ===
def translate_files(
    file_paths: list[str],
    source_language: str,
    source_language_full: str,
    target_languages: list[str],
    target_language_full: list[str],
    write_to_existing_files: bool = False
) -> dict[str, str]:
    """
    使用本地 vLLM 服务翻译文件内容。
    - 如果 file_paths 只有一个文件，则为完全翻译。
    - 如果 file_paths 有多个文件（[源文件, 目标文件1, 目标文件2...]），则为增量翻译。
    """
    if not file_paths:
        return {}

    # --- 1. 定义配置并确定模式 ---
    api_url = "http://192.168.5.2:31049/v1/chat/completions"
    model_path = "Qwen3-14B"
    is_incremental = len(file_paths) > 1
    
    if is_incremental and len(file_paths) != len(target_languages) + 1:
        raise ValueError("增量翻译模式下，文件路径数量必须等于目标语言数量 + 1")

    # --- 2. 读取源文件内容 ---
    source_file_path = file_paths[0]
    try:
        with open(source_file_path, 'r', encoding='utf-8') as f:
            source_content = f.read()
        print(f"[+] 成功读取源文件: {source_file_path}")
    except FileNotFoundError:
        print(f"[-] 错误：找不到源文件 {source_file_path}")
        return
    except Exception as e:
        print(f"[-] 读取源文件时发生错误: {e}")
        return {}

    # --- 3. 循环遍历所有目标语言进行翻译 ---
    final_results = {}
    for i, (lang_code, lang_full) in enumerate(zip(target_languages, target_language_full)):
        print(f"\n[*] --- 正在翻译为: {lang_full} ({lang_code}) ... ---")

        # --- 新增逻辑: 为增量模式加载现有目标文件内容 ---
        existing_target_content = None
        if is_incremental:
            target_file_path = file_paths[i + 1]
            try:
                with open(target_file_path, 'r', encoding='utf-8') as f:
                    existing_target_content = f.read()
                print(f"[i] 增量模式: 成功加载现有目标文件: {target_file_path}")
            except FileNotFoundError:
                print(f"[!] 增量模式警告: 未找到目标文件 {target_file_path}，将对其执行完全翻译。")

            # --- 重试机制 ---
        for attempt in range(1, MAX_RETRIES + 1):
            try:
                is_json_file = source_file_path.lower().endswith('.json')
                if is_json_file:
                    # --- 策略 A: 处理 JSON 文件 ---
                    source_data = json.loads(source_content)
                    existing_target_data = json.loads(existing_target_content) if existing_target_content else None
                    
                    translated_json = _translate_json_chunks(
                        source_data, source_language_full, lang_full, 
                        api_url, model_path,
                        existing_target_data=existing_target_data # 传入已存在的数据
                    )
                    original_ordered_keys = json.loads(source_content, object_pairs_hook=OrderedDict).keys()
                    
                    # 步骤 4: 根据原始 key 的顺序，构建一个新的有序字典
                    final_ordered_json = OrderedDict()
                    for key in original_ordered_keys:
                        # 从可能无序的翻译结果中，找到对应 key 的 value
                        if key in translated_json:
                            final_ordered_json[key] = translated_json[key]
                    
                    # 步骤 5: 将最终排序好的字典转换为 JSON 字符串
                    final_results[lang_code] = json.dumps(final_ordered_json, ensure_ascii=False, indent=2)
                else:
                    # --- 策略 B: 处理普通文本/MDX 文件 ---
                    translated_text = _translate_plain_text(
                        source_content, source_language_full, lang_full,
                        api_url, model_path,
                        existing_target_content=existing_target_content # 传入已存在的内容
                    )
                    if translated_text:
                        translated_text = translated_text.replace(f'/{source_language}/', f'/{lang_code}/')
                    if translated_text.endswith('---'):
                        translated_text += '\n'
                    
                    # 正则匹配开头 ---\ntitle: ... \n---\n 这种格式 不存在则报错
                    front_matter_pattern = re.compile(r'^---\s*\n(.*?)title:.*?\n---\s*\n', re.DOTALL)
                    if not front_matter_pattern.match(translated_text):
                        # 如果匹配失败 (返回 None)，则抛出一个 ValueError 异常，并提供清晰的错误信息。
                        raise ValueError(f"译文缺少有效的 Front Matter（例如 '--- title: ... ---'）。")
    
                    final_results[lang_code] = translated_text

                if is_incremental and write_to_existing_files:
                    target_file_path = file_paths[i + 1]
                    with open(target_file_path, 'w', encoding='utf-8') as f:
                        f.write(final_results[lang_code])
                    print(f"[+] 已写入翻译结果到现有文件: {target_file_path}")
                print(f"[+] 翻译成功: {lang_full} ({lang_code})")
                break # 成功则跳出重试循环
            except Exception as e:
                print(f"[-] 翻译尝试 {attempt} 失败: {e}")
                if attempt < MAX_RETRIES:
                    print("[i] 正在重试...")
                    time.sleep(2)
                    continue
                else:
                    print(f"[-] 在翻译到 {lang_full} 时发生严重错误，跳过该语言: {e}")
                    

    return final_results

# === 辅助函数: 策略 A - 翻译 JSON ===
def _translate_json_chunks(
    source_data: dict, 
    source_lang_full: str, 
    target_lang_full: str, 
    api_url: str, 
    model_path: str,
    chunk_size: int = 50,
    existing_target_data: dict | None = None # 接收现有数据
) -> dict:
    seperator = '...'
    
    # 步骤 1: 将源 JSON 展平
    print("[i] 步骤 1/3: 正在将嵌套的 JSON 展平...")
    flat_source_data = _flatten_json(source_data, separator=seperator)
    
    items_to_translate_all = {k: v for k, v in flat_source_data.items() if isinstance(v, (str, list))}
    other_items = {k: v for k, v in flat_source_data.items() if not isinstance(v, (str, list))}
    
    # --- 新增增量逻辑 ---
    if existing_target_data:
        print("[i] 检测到现有翻译，启用增量模式...")
        flat_target_data = _flatten_json(existing_target_data, separator=seperator)
        missing_keys = set(items_to_translate_all.keys()) - set(flat_target_data.keys())
        items_to_translate = {key: items_to_translate_all[key] for key in missing_keys}
        # 保留已有的翻译
        other_items.update(flat_target_data)
        print(f"[i] 需要新增翻译 {len(missing_keys)} 个条目。")
    else:
        items_to_translate = items_to_translate_all


    flat_translated_results = {}
    chunk = {}
    items = list(items_to_translate.items())

    system_prompt = (
        f"你是一个专业的、精通多种语言的翻译引擎。"
        f"你的任务是准确地将以下 JSON 对象中的所有**值 (value)** 从 {source_lang_full} 翻译成 {target_lang_full}。"
        f"请严格保持所有的**键 (key)** 和 JSON 结构不变。"
        f"只返回翻译后的 JSON 对象，不要添加任何额外的解释或代码块标记。"
    )
    
    print(f"[i] 步骤 2/3: 开始对 {len(items)} 个字符串条目进行分块翻译...")
    if not items:
        print("[i] 没有需要翻译的新条目。")
    
    for i, (key, value) in enumerate(items):
        chunk[key] = value

        if (len(chunk) >= chunk_size or (i == len(items) - 1 and chunk)):
            print(f"[i]   - 正在翻译块 (总进度: {i+1}/{len(items)})...")
            
            chunk_json_str = json.dumps(chunk, ensure_ascii=False, indent=2)
            user_prompt = f"{chunk_json_str} /no_think"

            translated_chunk_str = _call_llm_api(system_prompt, user_prompt, api_url, model_path)
            
            if translated_chunk_str:
                try:
                    translated_chunk = json.loads(translated_chunk_str)
                    flat_translated_results.update(translated_chunk)
                except json.JSONDecodeError:
                    print(f"[-] 警告：模型返回的 JSON 块格式无效，已跳过。返回内容:\n{translated_chunk_str}")
            else:
                print(f"[-] 警告：翻译失败，已跳过该块。")
            
            chunk = {}
            time.sleep(0.5)

    # 步骤 3: 将翻译好的字符串和其他类型的项合并，然后重建
    print("[i] 步骤 3/3: 正在将翻译结果重建为原始嵌套结构...")
    final_flat_data = {**other_items, **flat_translated_results}
    nested_translated_json = _unflatten_json(final_flat_data, separator=seperator)
    
    return nested_translated_json

# === 辅助函数: 策略 B - 翻译普通文本 ===
def _translate_plain_text(
    source_content: str, 
    source_lang_full: str, 
    target_lang_full: str, 
    api_url: str, 
    model_path: str,
    existing_target_content: str | None = None # 接收现有内容
) -> str:
    # --- 新增增量逻辑: 根据是否存在旧内容，选择不同的提示词 ---
    if existing_target_content:
        print("[i] 检测到现有翻译，为MDX/文本文件启用增量更新模式。")
        system_prompt = (
            f"你是一个专业的、精通多种语言的翻译引擎，擅长处理文档更新。"
            f"你的任务是：参考一份旧的 {target_lang_full} 翻译，将一份新的 {source_lang_full} 文档更新并翻译成 {target_lang_full}。"
            f"请仔细比对新旧源文的差异，并在旧译文的基础上进行修改，以最小的变动完成更新，同时保持翻译风格和术语的一致性。"
            f"严格保持原始文档的格式，如 Markdown 语法、换行和段落结构。"
            f"只返回最终完整的、更新后的 {target_lang_full} 译文，不要添加任何额外的解释或评论。"
        )
        user_prompt = (
            f"这是最新的 {source_lang_full} 文档内容：\n"
            f"--- [START OF NEW SOURCE] ---\n"
            f"{source_content}\n"
            f"--- [END OF NEW SOURCE] ---\n\n"
            f"这是之前对应的旧版 {target_lang_full} 翻译，请以此为基础进行更新：\n"
            f"--- [START OF OLD TRANSLATION] ---\n"
            f"{existing_target_content}\n"
            f"--- [END OF OLD TRANSLATION] ---\n"
            f"/no_think"
        )
    else:
        # 原始的完全翻译提示词
        system_prompt = (
            f"你是一个专业的、精通多种语言的翻译引擎。"
            f"你的任务是准确地将以下文本从 {source_lang_full} 翻译成 {target_lang_full}。"
            f"请严格保持原始文本的格式，如 Markdown 语法、换行和段落结构。"
            f"不要添加任何额外的解释、评论或与翻译内容无关的文字。"
        )
        user_prompt = f"{source_content} /no_think"
    
    translated_text = _call_llm_api(system_prompt, user_prompt, api_url, model_path)
    return translated_text if translated_text else ""


# === 主执行块 (已修改以演示新功能) ===
if __name__ == "__main__":
    try:

        # --- 示例 1: 更新现有的翻译文件 ---
        print("\n" + "="*20 + " 示例 1: 更新现有的翻译文件 " + "="*20)
        full_trans_results = translate_files(
            file_paths=[
                './zhCN/translation.json',  # 源文件（中文）
                './enUS/translation.json',  # 现有的英文翻译文件
                './ja/translation.json',   # 现有的日文翻译文件
                './ko/translation.json'    # 现有的韩文翻译文件
            ],
            source_language='zhCN',
            source_language_full='简体中文',
            target_languages=['enUS', 'ja', 'ko'],
            target_language_full=['English', '日本語', '한국어'],
            write_to_existing_files=True # 启用写回现有文件
        )



        # --- 示例 2: 创建新的翻译文件 ---
        source_file = 'example_data/source_zh.mdx'
        target_files = [
            'example_data/source_zh.en.mdx',
            'example_data/source_zh.ja.mdx'
        ]
        for tf in target_files:
            if os.path.exists(tf):
                os.remove(tf)
                print(f"[i] 删除现有目标文件以演示创建新文件: {tf}")
                with open(tf, 'w', encoding='utf-8') as f:
                    if tf.endswith('.json'):
                        f.write('{}')
        print("\n" + "="*20 + " 示例 2: 创建新的翻译文件 " + "="*20)
        full_trans_results = translate_files(
            file_paths=[source_file],
            source_language='zh',
            source_language_full='简体中文',
            target_languages=['en', 'ja'],
            target_language_full=['English', '日本語'],
            write_to_existing_files=True
        )


    except Exception as e:
        print(f"\n❌ 客户端执行时发生未处理的错误: {e}")
        import traceback
        traceback.print_exc()