// TODO: https://github.com/suren-atoyan/monaco-react?tab=readme-ov-file#use-monaco-editor-as-an-npm-package
import { useTheme } from "@/utils/theme";
import { Editor } from "@monaco-editor/react";

interface DockerfileEditorProps {
  value: string;
  onChange: (value: string | undefined) => void;
}

export function DockerfileEditor({ value, onChange }: DockerfileEditorProps) {
  const { theme } = useTheme();
  return (
    <Editor
      height="450px"
      defaultLanguage="dockerfile"
      value={value}
      onChange={onChange}
      theme={theme === "light" ? "light" : "vs-dark"}
      options={{
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        fontSize: 13,
        lineNumbers: "on",
        renderLineHighlight: "all",
        tabSize: 2,
      }}
      className="bg-transparen border-input overflow-hidden rounded-md border"
    />
  );
}
