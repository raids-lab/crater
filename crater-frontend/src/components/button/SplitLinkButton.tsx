import { PlusCircleIcon } from "lucide-react";
import { useNavigate } from "react-router-dom";
import SplitButton, { SplitButtonItem } from "./SplitButton";

interface URL {
  url: string;
  name: string;
  disabled?: boolean;
}

const SplitLinkButton = ({ urls, title }: { urls: URL[]; title: string }) => {
  const navigate = useNavigate();

  const items: SplitButtonItem[] = urls.map((url) => ({
    key: url.url,
    title: url.name,
    action: () => navigate(`/${url.url}`),
    disabled: url.disabled,
  }));

  return (
    <SplitButton
      icon={<PlusCircleIcon className="size-4" />}
      renderTitle={(title) => `新建${title}`}
      itemTitle="作业类型"
      items={items}
      cacheKey={title}
    />
  );
};

export default SplitLinkButton;
