import { defineConfig } from "vite";
import path from "path";
import react from "@vitejs/plugin-react-swc";
import inspect from "vite-plugin-inspect";
import { cdn } from "vite-plugin-cdn2";

export default defineConfig(({ command }) => {
  return {
    plugins: [
      react(),
      cdn({
        modules: [
          { name: "react", relativeModule: "./umd/react.production.min.js" },
          {
            name: "react-dom",
            relativeModule: "./umd/react-dom.production.min.js",
            aliases: ["client"],
          },
        ],
        apply: command,
      }),
      inspect(),
    ],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
  };
});
