import { CodeBlock } from "react-code-block";
import { themes } from "prism-react-renderer";
import { useTheme } from "@/utils/theme";

const BaseCodeBlock = ({
  ref,
  code,
  language,
  children,
}: { code: string; language: string; children?: React.ReactNode } & {
  ref?: React.RefObject<HTMLDivElement>;
}) => {
  const { theme } = useTheme();

  return (
    <CodeBlock
      code={code}
      language={language}
      theme={theme == "light" ? themes.vsLight : themes.vsDark}
    >
      <div className="relative" ref={ref}>
        {children}
        <CodeBlock.Code className="rounded-xl px-3 py-5 shadow-lg">
          <div className="table-row">
            <CodeBlock.LineNumber className="text-highlight-gray table-cell pr-4 text-right text-xs select-none" />
            <CodeBlock.LineContent className="table-cell text-sm break-words whitespace-pre-wrap">
              <CodeBlock.Token />
            </CodeBlock.LineContent>
          </div>
        </CodeBlock.Code>
      </div>
    </CodeBlock>
  );
};
BaseCodeBlock.displayName = "BaseCodeBlock";

export default BaseCodeBlock;
