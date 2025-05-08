"""
📚 Translation Bot – Powered by OpenAI 🌍
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
    python i18n-bot.py
"""



import os
import json
from openai import OpenAI
import glob
from pathlib import Path
from typing import Tuple

# === CONFIGURATION ===


PROJECT_ROOT = "."
TRANSLATION_PATH = Path("messages/en.json")
TARGET_FOLDERS = ["app", "components"]

# === CONFIGURATION ===
# API Configuration
API_BASE_URL = os.getenv("API_BASE_URL", "https://api.openai.com/v1")
API_KEY = os.getenv("API_KEY")
MODEL_NAME = os.getenv("MODEL_NAME", "gpt-4-1106-preview")
# Project Configuration
PROJECT_ROOT = "." 
TRANSLATION_PATH = Path("messages/en.json")
TARGET_FOLDERS = ["app", "components"]
client = OpenAI(
    api_key=API_KEY,
    base_url=API_BASE_URL
)

if not OPENAI_API_KEY:
    raise EnvironmentError("Please set your OPENAI_API_KEY environment variable.")


# === OPENAI REQUEST ===
def call_openai_for_i18n(code: str) -> Tuple[str, dict]:
    prompt = f"""
Do the following to the following React component code:

1. Import `useTranslations` from 'next-intl'.
2. Initialize `const t = useTranslations();` inside the component.
3. Wrap all user-visible text with `{{t('Your text here')}}`.
4. Dont refactor the code, dont delete comments. Just do the translation stuff.
5. Avoid using dots or commas in the translation keys.
6. Return:
   - The updated code with `t(...)` included
   - A flat JSON object of all the translation keys and values used, like:
     {{
       "hellow_world": "Hello world!",
       "click_here": "Click here",
       "loading": "Loading...",
     }}

Only return valid JavaScript/TypeScript code followed by the JSON.
Do not include triple backticks (```), markdown formatting, or labels like "JSON".
"""

    response = client.chat.completions.create(
        model=MODEL_NAME,
        messages=[
            {"role": "system", "content": "You are a helpful assistant for internationalizing vite/react apps."},
            {"role": "user", "content": prompt + "\n\n" + code}
        ]
    )

    content = response.choices[0].message.content

    # Strip markdown code blocks
    content = content.replace("```ts", "").replace("```tsx", "").replace("```json", "").replace("```", "").strip()

    # Split the content: first part is code, last part is JSON
    last_brace_index = content.rfind("{")
    if last_brace_index == -1:
        raise ValueError("No JSON object found in OpenAI response.")

    code_part = content[:last_brace_index].strip()
    json_part = content[last_brace_index:].strip()

    try:
        translations = json.loads(json_part)
    except json.JSONDecodeError as e:
        print("⚠️ Failed to parse JSON. Raw content:")
        print(json_part)
        raise e

    return code_part, translations


# === TRANSLATION UPDATE ===
def update_translation_json(new_keys: dict):
    if TRANSLATION_PATH.exists():
        with open(TRANSLATION_PATH, "r", encoding="utf-8") as f:
            existing = json.load(f)
    else:
        existing = {}

    merged = {**existing, **new_keys}
    with open(TRANSLATION_PATH, "w", encoding="utf-8") as f:
        json.dump(merged, f, indent=2, ensure_ascii=False)

# === PROCESS EACH FILE ===
def process_file(filepath: str):
    print(f"🔄 Processing: {filepath}")
    with open(filepath, "r", encoding="utf-8") as f:
        original_code = f.read()

    try:
        new_code, translations = call_openai_for_i18n(original_code)
    except Exception as e:
        print(f"❌ Error processing {filepath}: {e}")
        return

    with open(filepath, "w", encoding="utf-8") as f:
        f.write(new_code)

    print(f"✅ File overwritten: {filepath}")
    update_translation_json(translations)
    print(f"🌍 Translations updated.\n")

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
    print("🚀 Starting i18n transformation...\n")
    files = find_all_tsx_jsx_files()
    print(f"📁 Found {len(files)} files in 'app' and 'components'\n")

    for file in files:
        process_file(file)

    print("🎉 Done! All files processed and translation.json updated.")

if __name__ == "__main__":
    main()