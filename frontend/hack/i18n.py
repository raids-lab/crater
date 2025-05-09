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
import subprocess
from openai import OpenAI
import glob
from pathlib import Path
from typing import Tuple, Optional, Dict, Any, List

# === CONFIGURATION ===
# API Configuration
API_BASE_URL = os.getenv("API_BASE_URL", "http://192.168.5.22:31444/v1")
API_KEY = os.getenv("API_KEY", "placeholder")
MODEL_NAME = os.getenv("MODEL_NAME", "/models/Qwen3-32B")
MAX_RETRIES = 3
RETRY_DELAY = 2  # seconds

# Project Configuration
PROJECT_ROOT = "./src" 
TRANSLATION_PATH = Path("i18n/locales/zhCN/translation.json")
TARGET_FOLDERS = ["components/form", "components/custom", "pages"]
SCRIPT_VERSION = "1.1.0"  # Update this when making significant changes

client = OpenAI(
    api_key=API_KEY,
    base_url=API_BASE_URL
)

# 存储格式化失败的文件
formatting_failures = []

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

def extract_code_and_json(full_content: str) -> Optional[Tuple[str, Dict[str, str]]]:
    """Improved extraction of code and JSON parts from model response"""
    # First try the original method
    last_brace_index = full_content.rfind("{")
    if last_brace_index != -1:
        code_part = full_content[:last_brace_index].strip()
        json_part = full_content[last_brace_index:].strip()
        
        # Try parsing the JSON part
        try:
            translations = parse_json_with_templates(json_part)
            return code_part, translations
        except json.JSONDecodeError:
            pass  # Try alternative methods
    
    # If original method failed, try more sophisticated approaches
    
    # Method 1: Look for the last occurrence of "{\n" which is more likely to be JSON start
    last_json_start = full_content.rfind("{\n")
    if last_json_start != -1:
        code_part = full_content[:last_json_start].strip()
        json_part = full_content[last_json_start:].strip()
        
        try:
            translations = parse_json_with_templates(json_part)
            return code_part, translations
        except json.JSONDecodeError:
            pass
    
    # Method 2: Look for the last occurrence of "{\"" (JSON with quoted key)
    last_json_key_start = full_content.rfind('{"')
    if last_json_key_start != -1:
        code_part = full_content[:last_json_key_start].strip()
        json_part = full_content[last_json_key_start:].strip()
        
        try:
            translations = parse_json_with_templates(json_part)
            return code_part, translations
        except json.JSONDecodeError:
            pass
    
    # Method 3: Try to find the outermost matching braces
    try:
        # Find the first opening brace that has a matching closing brace at the end
        stack = []
        start_index = -1
        for i, c in enumerate(full_content):
            if c == '{':
                if not stack:
                    start_index = i
                stack.append(c)
            elif c == '}':
                if stack:
                    stack.pop()
                    if not stack:  # Found matching outermost braces
                        code_part = full_content[:start_index].strip()
                        json_part = full_content[start_index:i+1].strip()
                        
                        translations = parse_json_with_templates(json_part)
                        return code_part, translations
    except (json.JSONDecodeError, IndexError):
        pass
    
    # If all methods fail
    return None

def parse_json_with_templates(json_str: str) -> Dict[str, str]:
    """Parse JSON while handling template strings ({{ }})"""
    # Replace template markers temporarily
    template_placeholder_open = "__TEMPLATE_OPEN__"
    template_placeholder_close = "__TEMPLATE_CLOSE__"
    temp_json = json_str.replace("{{", template_placeholder_open).replace("}}", template_placeholder_close)
    
    # Parse the JSON
    translations_temp = json.loads(temp_json)
    
    # Restore template markers
    translations = {}
    for key, value in translations_temp.items():
        if isinstance(value, str):
            translations[key] = value.replace(template_placeholder_open, "{{").replace(template_placeholder_close, "}}")
        else:
            translations[key] = value
    
    return translations

# === OPENAI REQUEST ===
def call_openai_for_i18n(code: str) -> Optional[Tuple[str, Dict[str, str], bool]]:
    prompt = f"""
You are an expert React/i18n developer helping internationalize a React application using react-i18next. Follow these rules STRICTLY:

=== FIRST STEP: FILE ANALYSIS ===
Determine if the file contains ANY user-facing strings that need internationalization:
- User-facing strings include: UI text, button labels, error messages, tooltips, placeholders, etc.
- Technical strings to ignore: variable names, CSS classes, import paths, console logs, comments

If the file contains NO user-facing strings that need translation:
- Return only the exact string "NO_STRINGS_TO_TRANSLATE" without any other text
- Do not return the original code
- Do not add any import statements

=== COMPONENT RULES (if strings need translation) ===
1. Functional Components:
   - Add import: `import {{ useTranslation }} from "react-i18next";`
   - Initialize hook INSIDE component: `const {{ t }} = useTranslation();`
   - Place hook after all destructured props but before any other logic

2. Special Cases:
   - For Zod schemas: Move string literals into component ABOVE any form declarations
   - For constants: Convert to functions that accept t as parameter
   - For external configs: Create i18n wrapper functions

=== TRANSLATION RULES (if strings need translation) ===
1. Key Naming:
   - Use camelCase (e.g. "submitButton")
   - Follow hierarchy: [component].[element].[action] (e.g. "loginForm.emailLabel")
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
If file contains NO user-facing strings: return ONLY "NO_STRINGS_TO_TRANSLATE"

If file DOES contain user-facing strings, return EXACTLY:
1. The fully modified code with:
   - Correct imports
   - Proper t() usage
   - All original functionality preserved
   - Zod schemas moved to appropriate locations

2. A FLAT translations JSON object with:
   - All new key-value pairs
   - No nested structures
   - No duplicate keys

Example Output (with user-facing strings):
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

            # Remove all thinking content from full_content
            if "</think>" in full_content:
                full_content = full_content.replace(full_content[full_content.index("<think>"):full_content.index("</think>") + len("</think>")], "")

            # Check for the special marker indicating no strings to translate
            full_content = full_content.strip()
            if "NO_STRINGS_TO_TRANSLATE" in full_content:
                print("\n✅ File contains no user-facing strings to translate")
                return None, {}, True  # Return with a flag indicating no translation needed

            full_content = full_content.replace("```ts", "").replace("```tsx", "").replace("```json", "").replace("```", "").strip()

            result = extract_code_and_json(full_content)
            if result:
                code_part, translations = result
                return code_part, translations, False  # Return with flag indicating translation was performed
            else:
                print("\n⚠️ Failed to extract code and JSON from response")
                if attempt < MAX_RETRIES - 1:
                    print("Retrying...")
                    time.sleep(RETRY_DELAY)
                else:
                    raise ValueError("Could not extract valid code and JSON after multiple attempts")
        
        except Exception as e:
            print(f"\n⚠️ Attempt {attempt+1}/{MAX_RETRIES} failed: {str(e)}")
            if attempt < MAX_RETRIES - 1:
                print(f"Waiting {RETRY_DELAY} seconds before retry...")
                time.sleep(RETRY_DELAY)
            else:
                print("❌ Max retries reached, unable to process this file.")
                return None, {}, False
    
    return None, {}, False

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

# === FORMAT FILE ===
def format_with_prettier(filepath: str) -> bool:
    """Format file with Prettier"""
    try:
        print(f"🔍 Running Prettier on: {filepath}")
        # 确保相对于项目根目录运行命令
        project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
        result = subprocess.run(
            ["pnpm", "dlx", "prettier", "--write", os.path.join(PROJECT_ROOT, filepath)],
            cwd=project_root,
            capture_output=True,
            text=True,
            check=False
        )
        
        if result.returncode != 0:
            print(f"⚠️ Prettier formatting failed: {result.stderr}")
            formatting_failures.append(filepath)
            return False
        return True
    except Exception as e:
        print(f"⚠️ Error running Prettier: {e}")
        formatting_failures.append(filepath)
        return False

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
        
    new_code, translations, no_strings_flag = result
    
    if no_strings_flag:
        # Just add version comment without modifying the code
        version_comment = f"// i18n-processed-v{SCRIPT_VERSION} (no translatable strings)\n"
        if original_code.startswith("// i18n-processed-v"):
            # Replace existing version comment
            lines = original_code.splitlines()
            lines[0] = version_comment
            new_code = "\n".join(lines)
        else:
            new_code = version_comment + original_code
            
        with open(filepath, "w", encoding="utf-8") as f:
            f.write(new_code)
            
        print(f"\n✅ File marked as processed (no translatable strings): {filepath}")
        # 格式化已修改的文件
        format_with_prettier(filepath)
        return
    
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
    
    # 格式化已修改的文件
    format_with_prettier(filepath)

# === FILE SCANNER ===
def find_all_tsx_jsx_files():
    files = []
    # cd PROJECT_ROOT
    os.chdir(PROJECT_ROOT)
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
    
    # 输出格式化失败的文件列表
    if formatting_failures:
        print("\n⚠️ The following files failed Prettier formatting:")
        for failed_file in formatting_failures:
            print(f"  - {failed_file}")
        print(f"\nTotal formatting failures: {len(formatting_failures)}")
    else:
        print("\n✨ All files successfully formatted with Prettier!")

if __name__ == "__main__":
    main()