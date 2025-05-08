"""
📚 Translation Bot – Powered by Qwen3 🌍
----------------------------------------

📦 1. Install Dependencies
Run this in your terminal:
    python3 -m venv .venv
    source .venv/bin/activate
    pip install openai

🔑 2. Set Your OpenAI API Key
Export your key as an environment variable:
    export OPENAI_API_KEY=your-api-key-here

🚀 3. Run the Script
In the root of your project (where this script lives), run:
    python i18n.py
"""

import os
import json
import time
import sys
from openai import OpenAI
import glob
from pathlib import Path
from typing import Tuple, Optional, Dict, Any

# === CONFIGURATION ===
# API Configuration
API_BASE_URL = os.getenv("API_BASE_URL", "http://192.168.5.14:32598/v1")
API_KEY = os.getenv("API_KEY", "placeholder")
MODEL_NAME = os.getenv("MODEL_NAME", "/models/Qwen3-32B")
MAX_RETRIES = 3
RETRY_DELAY = 2  # seconds

# Project Configuration
PROJECT_ROOT = "." 
TRANSLATION_PATH = Path("i18n/locales/zhCN/translation.json")
TARGET_FOLDERS = ["components/custom", "pages"]
SCRIPT_VERSION = "1.1.0"  # Update this when making significant changes

client = OpenAI(
    api_key=API_KEY,
    base_url=API_BASE_URL
)

# === VERSION CHECK ===
def has_current_version(filepath: str) -> bool:
    """Check if file already processed with current version"""
    with open(filepath, "r", encoding="utf-8") as f:
        first_line = f.readline().strip()
    return f"// i18n-processed-v{SCRIPT_VERSION}" in first_line

def add_version_comment(filepath: str, content: str) -> str:
    """Add version comment to file content"""
    version_comment = f"// i18n-processed-v{SCRIPT_VERSION}\n"
    if content.startswith("// i18n-processed-v"):
        # Replace existing version comment
        lines = content.splitlines()
        lines[0] = version_comment
        return "\n".join(lines)
    return version_comment + content

# === CONNECTION TEST ===
def test_model_connection() -> bool:
    """Test model connection"""
    print("🔄 Testing model connection...")
    try:
        response = client.chat.completions.create(
            model=MODEL_NAME,
            messages=[
                {"role": "system", "content": "You are a helpful assistant."},
                {"role": "user", "content": "Say 'Connection successful' if you can see this message."}
            ],
            max_tokens=20,
            stream=True
        )
        
        print("🔄 Receiving stream response...")
        full_response = ""
        for chunk in response:
            content = chunk.choices[0].delta.content
            if content:
                print(content, end="", flush=True)
                full_response += content
        
        if "Connection successful" in full_response:
            print("\n✅ Model connection successful!")
            return True
        else:
            print(f"\n⚠️ Unexpected model response: {full_response}")
            return False
    except Exception as e:
        print(f"\n❌ Model connection failed: {str(e)}")
        print(f"Current configuration:")
        print(f"  - API Base URL: {API_BASE_URL}")
        print(f"  - Model: {MODEL_NAME}")
        print("Please check your API key and connection settings.")
        return False

# === OPENAI REQUEST ===
def call_openai_for_i18n(code: str) -> Optional[Tuple[str, Dict[str, str]]]:
    prompt = f"""
You are an expert React/i18n developer helping internationalize a React application using react-i18next. Follow these rules STRICTLY:

=== COMPONENT RULES ===
1. Functional Components:
   - Add import: `import {{ useTranslation }} from 'react-i18next';` at top
   - Initialize hook INSIDE component: `const {{ t }} = useTranslation();`
   - Place hook after all destructured props but before any other logic

2. Special Cases:
   - For Zod schemas: Move string literals into component ABOVE any form declarations
   - For constants: Convert to functions that accept t as parameter
   - For external configs: Create i18n wrapper functions

=== TRANSLATION RULES ===
1. Key Naming:
   - Use camelCase (e.g. 'submitButton')
   - Follow hierarchy: [component].[element].[action] (e.g. 'loginForm.emailLabel')
   - Never include dynamic values in keys
   - Keep keys under 3 levels deep maximum

2. Text Handling:
   - Wrap ALL user-facing strings including:
     * JSX text content
     * Placeholders
     * Aria labels
     * Error messages
     * Button/text alternatives
   - Skip:
     * Console logs
     * Internal IDs
     * Technical error codes

3. Existing Code:
   - Preserve ALL comments and code structure
   - Never modify existing t() calls
   - Maintain original formatting/indentation

=== OUTPUT FORMAT ===
Return EXACTLY:
1. The fully modified code with:
   - Correct imports
   - Proper t() usage
   - All original functionality preserved
   - Zod schemas moved to appropriate locations

2. A FLAT translations JSON object with:
   - All new key-value pairs
   - No nested structures
   - No duplicate keys

Example Output:
// Modified code
import {{ useTranslation }} from 'react-i18next';

function LoginForm() {{
  const {{ t }} = useTranslation();
  
  // Moved Zod schema to component
  const loginSchema = z.object({{
    email: z.string().email(t('loginForm.emailError')),
    password: z.string().min(8, t('loginForm.passwordError'))
  }});
  
  const form = useForm({{ resolver: zodResolver(loginSchema) }});

  return (
    <button aria-label={{t('loginForm.submitAriaLabel')}}>
      {{t("loginForm.submitButton", {{ name: "John" }})}}
    </button>
  );
}}

{{
  "loginForm.submitButton": "Welcome {{{{name}}}}",
  "loginForm.submitAriaLabel": "Submit login form",
  "loginForm.emailError": "Invalid email address",
  "loginForm.passwordError": "Password must be at least 8 characters"
}}

=== IMPORTANT ===
- NEVER use markdown syntax (```) in response
- NEVER include explanatory text
- ALWAYS return valid code + JSON
- If unsure about a string, ASK via <think> tags
"""

    for attempt in range(MAX_RETRIES):
        try:
            print("\n🔄 Sending request to model...")
            response = client.chat.completions.create(
                model=MODEL_NAME,
                messages=[
                    {"role": "system", "content": "You are a helpful assistant for internationalizing React applications."},
                    {"role": "user", "content": prompt + "\n\nOriginal code:\n" + code}
                ],
                stream=True,
            )

            full_content = ""
            thinking_content = ""
            in_thinking = False
            
            print("🔄 Receiving stream response:")
            for chunk in response:
                delta = chunk.choices[0].delta
                if delta.content:
                    content = delta.content
                    print(content, end="", flush=True)
                    full_content += content
                    
                    if "<think>" in content:
                        in_thinking = True
                        thinking_content += content[content.index("<think>") + len("<think>"):]
                    elif "</think>" in content:
                        in_thinking = False
                        thinking_content += content[:content.index("</think>")]
                        print(f"\n🤔 Model thinking: {thinking_content}")
                        thinking_content = ""
                    elif in_thinking:
                        thinking_content += content

            # remove all thinking content from full_content
            if "</think>" in full_content:
                # remove everything between <think> and </think>
                full_content = full_content.replace(thinking_content, "")

            full_content = full_content.replace("```ts", "").replace("```tsx", "").replace("```json", "").replace("```", "").strip()

            last_brace_index = full_content.rfind("{")
            if last_brace_index == -1:
                print("\n⚠️ No JSON object found in response. Retrying...")
                if attempt < MAX_RETRIES - 1:
                    time.sleep(RETRY_DELAY)
                    continue
                else:
                    raise ValueError("No JSON object found in OpenAI response")

            code_part = full_content[:last_brace_index].strip()
            json_part = full_content[last_brace_index:].strip()

            try:
                # 在解析之前替换掉可能导致问题的模板字符串
                # 将模板字符串中的 {{ 和 }} 临时替换为特殊标记
                template_placeholder_open = "__TEMPLATE_OPEN__"
                template_placeholder_close = "__TEMPLATE_CLOSE__"
                temp_json_part = json_part.replace("{{", template_placeholder_open).replace("}}", template_placeholder_close)
                
                # 尝试解析修改后的JSON
                translations_temp = json.loads(temp_json_part)
                
                # 将结果中的临时标记还原为 {{ 和 }}
                translations = {}
                for key, value in translations_temp.items():
                    if isinstance(value, str):
                        translations[key] = value.replace(template_placeholder_open, "{{").replace(template_placeholder_close, "}}")
                    else:
                        translations[key] = value
                
                return code_part, translations
            except json.JSONDecodeError as e:
                print(f"\n⚠️ JSON parsing failed: {str(e)}")
                print("Raw JSON content:")
                print(json_part)
                
                # 尝试更复杂的修复方法
                if attempt < MAX_RETRIES - 1:
                    print("Retrying with different JSON extraction method...")
                    try:
                        # 尝试使用更严格的正则表达式提取JSON
                        import re
                        json_match = re.search(r'(\{[\s\S]*\})$', full_content)
                        if json_match:
                            better_json = json_match.group(1)
                            # 同样替换模板字符串
                            better_json = better_json.replace("{{", template_placeholder_open).replace("}}", template_placeholder_close)
                            translations_temp = json.loads(better_json)
                            
                            # 恢复模板标记
                            translations = {}
                            for key, value in translations_temp.items():
                                if isinstance(value, str):
                                    translations[key] = value.replace(template_placeholder_open, "{{").replace(template_placeholder_close, "}}")
                                else:
                                    translations[key] = value
                                    
                            return code_part, translations
                    except Exception as inner_e:
                        print(f"Advanced extraction failed: {inner_e}")
                        
                    print("Retrying with model...")
                    time.sleep(RETRY_DELAY)
                else:
                    raise e
                
        except Exception as e:
            print(f"\n⚠️ Attempt {attempt+1}/{MAX_RETRIES} failed: {str(e)}")
            if attempt < MAX_RETRIES - 1:
                print(f"Waiting {RETRY_DELAY} seconds before retry...")
                time.sleep(RETRY_DELAY)
            else:
                print("❌ Max retries reached, unable to process this file.")
                return None
    
    return None

# === TRANSLATION UPDATE ===
def update_translation_json(new_keys: Dict[str, str]):
    if TRANSLATION_PATH.exists():
        with open(TRANSLATION_PATH, "r", encoding="utf-8") as f:
            existing = json.load(f)
    else:
        existing = {}

    merged = {**new_keys, **existing}
    TRANSLATION_PATH.parent.mkdir(parents=True, exist_ok=True)
    with open(TRANSLATION_PATH, "w", encoding="utf-8") as f:
        json.dump(merged, f, indent=2, ensure_ascii=False)

# === PROCESS EACH FILE ===
def process_file(filepath: str):
    print(f"\n📄 Processing file: {filepath}")
    
    if has_current_version(filepath):
        print(f"⚠️ Skipping: File already processed with version {SCRIPT_VERSION}")
        return

    with open(filepath, "r", encoding="utf-8") as f:
        original_code = f.read()

    if original_code.startswith("// ignore-i18n-script"):
        print("⚠️ Skipping: File contains ignore comment.")
        return

    result = call_openai_for_i18n(original_code)
    if result is None:
        print(f"❌ Failed to process file {filepath}")
        return
        
    new_code, translations = result
    
    # Verify useTranslation placement
    if "useTranslation" in new_code:
        if "const { t } = useTranslation();" not in new_code:
            print("⚠️ Warning: useTranslation may not be properly initialized inside component")
    
    # Add version comment
    new_code = add_version_comment(filepath, new_code)
    
    with open(filepath, "w", encoding="utf-8") as f:
        f.write(new_code)

    print(f"\n✅ File updated: {filepath}")
    update_translation_json(translations)
    print(f"🌍 Added {len(translations)} new translations.")

# === FILE SCANNER ===
def find_all_tsx_jsx_files():
    files = []
    for folder in TARGET_FOLDERS:
        for ext in ("tsx", "ts", "jsx", "js"):
            for file in glob.glob(f"{folder}/**/*.{ext}", recursive=True):
                if not any(skip in file for skip in ["node_modules", "dist", ".venv"]):
                    files.append(file)
    return files

# === MAIN ENTRYPOINT ===
def main():
    print("\n🚀 Starting i18n conversion...")
    print(f"🔧 Configuration:")
    print(f"  - Model: {MODEL_NAME}")
    print(f"  - API Base URL: {API_BASE_URL}")
    print(f"  - Target folders: {TARGET_FOLDERS}")
    print(f"  - Translation file: {TRANSLATION_PATH}")
    print(f"  - Script version: {SCRIPT_VERSION}\n")
    
    if not test_model_connection():
        print("❌ Cannot connect to model, please check your configuration.")
        return
        
    files = find_all_tsx_jsx_files()
    print(f"\n📁 Found {len(files)} files to process.")

    for i, file in enumerate(files, 1):
        print(f"\n🔧 Processing progress: {i}/{len(files)}")
        process_file(file)

    print("\n🎉 Done! All files processed and translation file updated.")

if __name__ == "__main__":
    main()