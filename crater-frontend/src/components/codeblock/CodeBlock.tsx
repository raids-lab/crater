import { CodeBlock } from "react-code-block";
import { themes } from "prism-react-renderer";
import { useTheme } from "@/utils/theme";

function LogBlock({
  code,
  language,
}: {
  code: string;
  language: string;
  handleCopy?: () => void;
}) {
  const { theme } = useTheme();

  return (
    <CodeBlock
      code={code}
      language={language}
      theme={theme == "light" ? themes.vsDark : themes.vsDark}
    >
      <div className="relative">
        <CodeBlock.Code className="rounded-xl bg-gray-900 !p-6 shadow-lg">
          <div className="table-row">
            <CodeBlock.LineNumber className="table-cell select-none pr-4 text-right text-sm text-gray-500" />
            <CodeBlock.LineContent className="table-cell">
              <CodeBlock.Token />
            </CodeBlock.LineContent>
          </div>
        </CodeBlock.Code>
      </div>
    </CodeBlock>
  );
}

export default LogBlock;
