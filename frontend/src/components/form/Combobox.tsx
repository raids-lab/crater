import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form";
import { cn } from "@/lib/utils";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Check, ChevronsUpDown } from "lucide-react";
// import { ScrollArea } from "@/components/ui/scroll-area";
import { useState } from "react";
import { ScrollArea } from "../ui/scroll-area";

export interface ComboboxItem<T> {
  label: string;
  value: string;
  detail?: T;
}

type ComboboxProps<T> = React.HTMLAttributes<HTMLDivElement> & {
  formTitle: string;
  items: ComboboxItem<T>[];
  current: string;
  disabled?: boolean;
  handleSelect: (value: string) => void;
  renderLabel?: (item: ComboboxItem<T>) => React.ReactNode;
  className?: string;
};

function Combobox<T>({
  formTitle,
  items,
  current,
  disabled,
  handleSelect,
  renderLabel,
  className,
}: ComboboxProps<T>) {
  const [open, setOpen] = useState(false);
  return (
    <Popover modal open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <FormControl>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className={cn(
              "w-full justify-between px-3 font-normal text-ellipsis whitespace-nowrap",
              "data-[state=open]:ring-ring data-[state=open]:ring-1 data-[state=open]:outline-hidden",
              !current && "text-muted-foreground",
              className,
            )}
            disabled={disabled}
          >
            <p className="truncate">
              {current
                ? items.find((dataset) => dataset.value === current)?.label
                : `选择${formTitle}`}
            </p>
            <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
          </Button>
        </FormControl>
      </PopoverTrigger>
      <PopoverContent
        className="z-50 p-0"
        style={{
          width: "var(--radix-popover-trigger-width)",
          maxHeight: "var(--radix-popover-content-available-height)",
        }}
      >
        <Command>
          <CommandInput placeholder={`查找${formTitle}`} className="h-9" />
          <CommandList>
            <CommandEmpty>未找到匹配的{formTitle}</CommandEmpty>
            <CommandGroup>
              <ScrollArea>
                <div className="max-h-48">
                  {items.map((item) => (
                    <CommandItem
                      value={item.label}
                      key={item.value}
                      onSelect={() => {
                        handleSelect(item.value);
                        setOpen(false);
                      }}
                      className="flex w-full flex-row items-center justify-between"
                    >
                      {renderLabel ? renderLabel(item) : <>{item.label}</>}
                      <Check
                        className={cn(
                          "ml-auto size-4",
                          item.value === current ? "opacity-100" : "opacity-0",
                        )}
                      />
                    </CommandItem>
                  ))}
                </div>
              </ScrollArea>
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

export default Combobox;
