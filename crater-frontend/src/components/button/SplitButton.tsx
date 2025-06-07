import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { ChevronDownIcon } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import { useLocalStorage } from "usehooks-ts";
import { ReactNode, useEffect } from "react";

export interface SplitButtonItem {
  key: string;
  title: string;
  action: () => void;
  disabled?: boolean;
}

interface SplitButtonProps {
  icon: ReactNode;
  renderTitle: (itemTitle?: string) => ReactNode;
  itemTitle: string;
  items: SplitButtonItem[];
  cacheKey: string;
}

const SplitButton = ({
  icon,
  renderTitle,
  itemTitle,
  items,
  cacheKey,
}: SplitButtonProps) => {
  const [position, setPosition] = useLocalStorage(
    `split-button-${cacheKey}`,
    items[0].key,
  );

  useEffect(() => {
    // if position is not in items, or item is disabled, set position to first not disabled
    if (
      !items.find((item) => item.key === position) ||
      items.find((item) => item.key === position)?.disabled
    ) {
      setPosition(items.find((item) => !item.disabled)?.key || items[0].key);
    }
  }, [items, position, setPosition]);

  return (
    <div className="bg-highlight hover:bg-highlight/90 flex h-9 w-fit items-center rounded-md transition-colors">
      <Button
        className="text-highlight-foreground hover:text-highlight-foreground cursor-pointer rounded-r-none pr-3 capitalize shadow-none hover:bg-transparent dark:hover:bg-transparent"
        variant="ghost"
        onClick={() => items.find((item) => item.key === position)?.action()}
      >
        {icon}
        {renderTitle(items.find((item) => item.key === position)?.title)}
      </Button>
      <Separator orientation="vertical" className="bg-background" />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            className="text-highlight-foreground cursor-pointer rounded-l-none pr-3 pl-2 shadow-none hover:bg-transparent focus:ring-0 dark:hover:bg-transparent"
            variant="ghost"
          >
            <ChevronDownIcon className="text-highlight-foreground size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          alignOffset={-5}
          className="w-[200px]"
          forceMount
        >
          <DropdownMenuLabel className="text-muted-foreground text-xs">
            {itemTitle}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuRadioGroup value={position} onValueChange={setPosition}>
            {items.map((item) => (
              <DropdownMenuRadioItem
                key={item.key}
                value={item.key}
                disabled={item.disabled}
              >
                {item.title}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};

export default SplitButton;
