import { CodeBlock } from "react-code-block";
import { themes } from "prism-react-renderer";
import { useTheme } from "@/utils/theme";

function BaseCodeBlock({ code, language }: { code: string; language: string }) {
  const { theme } = useTheme();

  return (
    <CodeBlock
      code={code}
      language={language}
      theme={theme == "light" ? themes.vsDark : themes.vsDark}
    >
      <div className="relative">
        <CodeBlock.Code className="rounded-xl px-3 py-5 shadow-lg">
          <div className="table-row">
            <CodeBlock.LineNumber className="table-cell select-none pr-4 text-right text-xs text-gray-500" />
            <CodeBlock.LineContent className="table-cell text-sm">
              <CodeBlock.Token />
            </CodeBlock.LineContent>
          </div>
        </CodeBlock.Code>
      </div>
    </CodeBlock>
  );
}

export default BaseCodeBlock;
