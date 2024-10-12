import { CodeBlock } from "react-code-block";
import { themes } from "prism-react-renderer";
import { useTheme } from "@/utils/theme";
import { forwardRef } from "react";

const BaseCodeBlock = forwardRef<
  HTMLDivElement,
  { code: string; language: string; children?: React.ReactNode }
>(({ code, language, children }, ref) => {
  const { theme } = useTheme();

  return (
    <CodeBlock
      code={code}
      language={language}
      theme={theme == "light" ? themes.vsDark : themes.vsDark}
    >
      <div className="relative" ref={ref}>
        {children}
        <CodeBlock.Code className="rounded-xl px-3 py-5 shadow-lg">
          <div className="table-row">
            <CodeBlock.LineNumber className="table-cell select-none pr-4 text-right text-xs text-gray-500" />
            <CodeBlock.LineContent className="table-cell whitespace-pre-wrap break-words text-sm">
              <CodeBlock.Token />
            </CodeBlock.LineContent>
          </div>
        </CodeBlock.Code>
      </div>
    </CodeBlock>
  );
});
BaseCodeBlock.displayName = "BaseCodeBlock";

export default BaseCodeBlock;
