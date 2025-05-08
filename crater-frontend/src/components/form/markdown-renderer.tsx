// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
import React, {
  JSX,
  Suspense,
  useMemo,
  useState,
  useEffect,
  useRef,
} from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { cn } from "@/lib/utils";
import { CopyButton } from "@/components/button/copy-button";
import { logger } from "@/utils/loglevel";
import { useTheme } from "@/utils/theme";

// 添加全局缓存对象来存储已处理的代码高亮结果
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const highlightCache = new Map<string, any[]>();

interface MarkdownRendererProps {
  children: string;
}

// 使用React.memo来避免不必要的重渲染
export const MarkdownRenderer = React.memo(
  ({ children }: MarkdownRendererProps) => {
    return (
      <div className="space-y-3">
        <Markdown remarkPlugins={[remarkGfm]} components={COMPONENTS}>
          {children}
        </Markdown>
      </div>
    );
  },
);
MarkdownRenderer.displayName = "MarkdownRenderer";

interface HighlightedPreProps extends React.HTMLAttributes<HTMLPreElement> {
  children: string;
  language: string;
  withLineNumbers?: boolean;
}

export const HighlightedPre = React.memo(
  ({ children, language, withLineNumbers, ...props }: HighlightedPreProps) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const [tokens, setTokens] = useState<any[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const isMountedRef = useRef(true);
    const { theme } = useTheme();

    // 创建唯一缓存键
    const cacheKey = useMemo(
      () => `${language}:${children}`,
      [language, children],
    );

    useEffect(() => {
      isMountedRef.current = true;

      // 检查缓存中是否已有结果
      if (highlightCache.has(cacheKey)) {
        setTokens(highlightCache.get(cacheKey) || []);
        setIsLoading(false);
        return;
      }

      const abortController = new AbortController();

      async function highlightCode() {
        try {
          // 使用动态导入并限制导入频率
          const { codeToTokens, bundledLanguages } = await import("shiki");

          if (!isMountedRef.current || abortController.signal.aborted) return;

          if (!(language in bundledLanguages)) {
            setIsLoading(false);
            return;
          }

          const result = await codeToTokens(children, {
            lang: language as keyof typeof bundledLanguages,
            defaultColor: false,
            themes: {
              light: "github-light",
              dark: "github-dark",
            },
          });

          if (isMountedRef.current && !abortController.signal.aborted) {
            // 存储结果到缓存
            highlightCache.set(cacheKey, result.tokens);
            setTokens(result.tokens);
            setIsLoading(false);
          }
        } catch (error) {
          logger.error("Code highlighting error:", error);
          if (isMountedRef.current && !abortController.signal.aborted) {
            setIsLoading(false);
          }
        }
      }

      // 使用requestIdleCallback延迟处理，减少CPU占用
      if (typeof window !== "undefined" && "requestIdleCallback" in window) {
        window.requestIdleCallback(() => {
          if (isMountedRef.current) highlightCode();
        });
      } else {
        // 降低优先级的setTimeout
        setTimeout(highlightCode, 0);
      }

      return () => {
        isMountedRef.current = false;
        abortController.abort();
      };
    }, [cacheKey, children, language]);

    if (isLoading) {
      return <pre {...props}>{children}</pre>;
    }

    if (tokens.length === 0) {
      return <pre {...props}>{children}</pre>;
    }

    return (
      <pre {...props}>
        <code>
          {tokens.map((line, lineIndex) => (
            <React.Fragment key={lineIndex}>
              <span>
                {withLineNumbers && (
                  <span className="text-highlight-slate/50 mr-2.5 select-none">
                    {String(lineIndex + 1).padStart(
                      String(tokens.length).length,
                      " ",
                    )}
                  </span>
                )}
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {line.map((token: any, tokenIndex: number) => {
                  const tokenStyle =
                    typeof token.htmlStyle === "string"
                      ? undefined
                      : token.htmlStyle;

                  const style = {
                    ...tokenStyle,
                    color:
                      theme === "light"
                        ? "var(--shiki-light)"
                        : "var(--shiki-dark)",
                  };

                  return (
                    <span
                      key={tokenIndex}
                      className="text-shiki-light bg-shiki-light-bg dark:text-shiki-dark dark:bg-shiki-dark-bg"
                      style={style}
                    >
                      {token.content}
                    </span>
                  );
                })}
              </span>
              {lineIndex !== tokens.length - 1 && "\n"}
            </React.Fragment>
          ))}
        </code>
      </pre>
    );
  },
);
HighlightedPre.displayName = "HighlightedCode";

interface CodeBlockProps extends React.HTMLAttributes<HTMLPreElement> {
  children: React.ReactNode;
  className?: string;
  language: string;
}

const ShikiCodeBlock = React.memo(
  ({ children, className, language, ...restProps }: CodeBlockProps) => {
    const { t } = useTranslation();
    const code = useMemo(() => {
      if (typeof children === "string") {
        return children;
      }
      return childrenTakeAllStringContents(children);
    }, [children]);

    const preClass = useMemo(
      () =>
        cn(
          "overflow-x-scroll rounded-md border bg-background/50 p-4 font-mono text-sm [scrollbar-width:none]",
          className,
        ),
      [className],
    );

    // 根据代码长度使用不同渲染策略
    const shouldUseLightRenderer = useMemo(
      () => code.length > 5000,
      [code.length],
    );

    if (shouldUseLightRenderer) {
      // 对于特别长的代码块使用简单渲染
      return (
        <div className="group/code relative mb-4">
          <pre className={preClass} {...restProps}>
            <code>{code}</code>
          </pre>
          <div className="invisible absolute top-2 right-2 flex space-x-1 rounded-lg p-1 opacity-0 transition-all duration-200 group-hover/code:visible group-hover/code:opacity-100">
            <CopyButton
              content={code}
              copyMessage={t("codeBlock.copyMessage")}
            />
          </div>
        </div>
      );
    }

    return (
      <div className="group/code relative mb-4">
        <Suspense
          fallback={
            <pre className={preClass} {...restProps}>
              {children}
            </pre>
          }
        >
          <HighlightedPre language={language} className={preClass}>
            {code}
          </HighlightedPre>
        </Suspense>

        <div className="invisible absolute top-2 right-2 flex space-x-1 rounded-lg p-1 opacity-0 transition-all duration-200 group-hover/code:visible group-hover/code:opacity-100">
          <CopyButton content={code} copyMessage={t("codeBlock.copyMessage")} />
        </div>
      </div>
    );
  },
);
ShikiCodeBlock.displayName = "CodeBlock";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const Code = React.memo(({ children, className, ...rest }: any) => {
  const match = /language-(\w+)/.exec(className || "");
  return match ? (
    <ShikiCodeBlock className={className} language={match[1]} {...rest}>
      {children}
    </ShikiCodeBlock>
  ) : (
    <code
      className={cn(
        "[:not(pre)>&]:bg-muted/50 font-mono [:not(pre)>&]:rounded-md [:not(pre)>&]:px-1 [:not(pre)>&]:py-0.5",
      )}
      {...rest}
    >
      {children}
    </code>
  );
});
Code.displayName = "Code";

function childrenTakeAllStringContents(element: unknown): string {
  // 处理 null 或 undefined
  if (element == null) {
    return "";
  }

  // 处理字符串直接返回
  if (typeof element === "string") {
    return element;
  }

  // 处理数组类型
  if (Array.isArray(element)) {
    return element
      .map((child) => childrenTakeAllStringContents(child))
      .join("");
  }

  // 处理 React 元素
  if (
    typeof element === "object" &&
    "props" in element &&
    element.props !== null &&
    typeof element.props === "object"
  ) {
    const props = element.props;
    if ("children" in props) {
      const children = props.children;

      if (Array.isArray(children)) {
        return children
          .map((child) => childrenTakeAllStringContents(child))
          .join("");
      } else {
        return childrenTakeAllStringContents(children);
      }
    }
  }

  // 处理其他情况，如数字
  return String(element || "");
}

const COMPONENTS = {
  h1: withClass("h1", "text-2xl font-semibold"),
  h2: withClass("h2", "font-semibold text-xl text-primary"),
  h3: withClass("h3", "font-semibold text-lg"),
  h4: withClass("h4", "font-semibold text-base"),
  h5: withClass("h5", "font-medium"),
  strong: withClass("strong", "font-semibold"),
  a: withClass("a", "text-primary underline underline-offset-2"),
  blockquote: withClass("blockquote", "border-l-2 border-primary pl-4"),
  code: Code,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  pre: ({ children }: any) => children,
  ol: withClass("ol", "list-decimal space-y-2 pl-6"),
  ul: withClass("ul", "list-disc space-y-2 pl-6"),
  li: withClass("li", "my-1.5"),
  table: withClass(
    "table",
    "w-full border-collapse overflow-y-auto rounded-md border border-foreground/20",
  ),
  th: withClass(
    "th",
    "border border-foreground/20 px-4 py-2 text-left font-bold [&[align=center]]:text-center [&[align=right]]:text-right",
  ),
  td: withClass(
    "td",
    "border border-foreground/20 px-4 py-2 text-left [&[align=center]]:text-center [&[align=right]]:text-right",
  ),
  tr: withClass("tr", "m-0 border-t p-0 even:bg-muted"),
  p: withClass("p", "whitespace-pre-wrap"),
  hr: withClass("hr", "border-foreground/20"),
};

function withClass(Tag: keyof JSX.IntrinsicElements, classes: string) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const Component = ({ ...props }: any) => (
    <Tag className={classes} {...props} />
  );
  Component.displayName = String(Tag);
  return Component;
}

export default MarkdownRenderer;
