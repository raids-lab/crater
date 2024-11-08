import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";
import pluginQuery from "@tanstack/eslint-plugin-query";
import eslintPluginPrettier from "eslint-plugin-prettier/recommended";

export default tseslint.config({
  extends: [
    js.configs.recommended,
    ...tseslint.configs.recommended,
    ...pluginQuery.configs["flat/recommended"],
    eslintPluginPrettier,
  ],
  files: ["**/*.{ts,tsx}"],
  ignores: [
    "**/dist",
    "**/postcss.config.js",
    "**/tailwind.config.js",
    "**/vite.config.ts",
    "**/commitlint.config.cjs",
    // shadcn
    "src/components/ui/*",
    "src/components/ui-custom/*",
    "src/hooks/use-mobile.tsx",
    // msw
    "public/mockServiceWorker.js",
  ],
  languageOptions: {
    ecmaVersion: 2020,
    globals: globals.browser,
  },
  plugins: {
    "react-hooks": reactHooks,
    "react-refresh": reactRefresh,
  },
  rules: {
    ...reactHooks.configs.recommended.rules,
    "react-refresh/only-export-components": [
      "off", // todo(liyilong): change to warn
      { allowConstantExport: true },
    ],
    "no-console": "error",
  },
});
