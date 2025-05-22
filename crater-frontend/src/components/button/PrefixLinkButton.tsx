import { ExternalLink } from "lucide-react";
import SplitButton, { SplitButtonItem } from "./SplitButton";
import { useMemo } from "react";

interface PrefixLinkButtonProps {
  names: string[];
  prefixes: string[];
  title?: string;
}

const PrefixLinkButton = ({
  names = [],
  prefixes = [],
  title = " ",
}: PrefixLinkButtonProps) => {
  // 用 useMemo 优化过滤和 items 构建
  const filtered = useMemo(
    () =>
      Array.isArray(names) && Array.isArray(prefixes)
        ? names
            .map((name, idx) => ({ name, prefix: prefixes[idx] }))
            .filter((item) => item.name !== "notebook")
        : [],
    [names, prefixes],
  );

  const items: SplitButtonItem[] = useMemo(
    () =>
      filtered.map((item) => ({
        key: item.prefix,
        title: item.name,
        action: () => window.open(item.prefix, "_blank"),
      })),
    [filtered],
  );

  if (!filtered || filtered.length === 0) {
    return null;
  }

  return (
    <SplitButton
      icon={<ExternalLink className="size-4" />}
      renderTitle={(title) => `${title}`}
      itemTitle="入口类型"
      items={items}
      cacheKey={title}
    />
  );
};

export default PrefixLinkButton;
