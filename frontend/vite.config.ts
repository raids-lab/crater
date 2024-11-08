import { defineConfig } from "vite";
import path from "path";
import react from "@vitejs/plugin-react-swc";
import inspect from "vite-plugin-inspect";

export default defineConfig({
  plugins: [react(), inspect()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
