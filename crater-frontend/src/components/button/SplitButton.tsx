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
    <div className="flex w-fit items-center space-x-1 rounded-md bg-primary text-primary-foreground">
      <Button
        className="pr-3 shadow-none"
        onClick={() => items.find((item) => item.key === position)?.action()}
      >
        {icon}
        {renderTitle(items.find((item) => item.key === position)?.title)}
      </Button>
      <Separator orientation="vertical" className="h-[20px]" />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button className="pl-2 pr-3 shadow-none focus:ring-0">
            <ChevronDownIcon className="size-4 text-primary-foreground" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          alignOffset={-5}
          className="w-[200px]"
          forceMount
        >
          <DropdownMenuLabel className="text-xs text-muted-foreground">
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
