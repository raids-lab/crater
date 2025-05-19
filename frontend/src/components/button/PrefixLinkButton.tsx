import { ExternalLink } from "lucide-react";
import SplitButton, { SplitButtonItem } from "./SplitButton";

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
  if (!names || !prefixes || names.length === 0 || prefixes.length === 0) {
    return null;
  }

  // 过滤掉 name 为 "notebook" 的规则
  const filtered = names
    .map((name, idx) => ({ name, prefix: prefixes[idx] }))
    .filter((item) => item.name !== "notebook");

  if (filtered.length === 0) {
    return null;
  }

  const items: SplitButtonItem[] = filtered.map((item) => ({
    key: item.prefix,
    title: item.name,
    action: () => window.open(item.prefix, "_blank"),
  }));

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
