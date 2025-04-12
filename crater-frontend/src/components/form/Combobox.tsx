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
  CommandDialog,
} from "@/components/ui/command";
import { Check, ChevronsUpDown } from "lucide-react";
// import { ScrollArea } from "@/components/ui/scroll-area";
import { useState } from "react";

export interface ComboboxItem<T> {
  label: string;
  value: string;
  selectedLabel?: string;
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
  useDialog?: boolean;
};

const getSelectedLabel = <T,>(
  items: ComboboxItem<T>[],
  current: string,
): string | undefined => {
  const selectedItem = items.find((item) => item.value === current);
  return selectedItem
    ? (selectedItem.selectedLabel ?? selectedItem.label)
    : undefined;
};

function Combobox<T>({
  formTitle,
  items,
  current,
  disabled,
  handleSelect,
  renderLabel,
  className,
  useDialog = false,
}: ComboboxProps<T>) {
  const [open, setOpen] = useState(false);

  const triggerButton = (
    <FormControl>
      <Button
        variant="outline"
        role="combobox"
        type="button"
        aria-expanded={open}
        aria-describedby=""
        className={cn(
          "w-full justify-between px-3 font-normal text-ellipsis whitespace-nowrap",
          "data-[state=open]:border-ring data-[state=open]:ring-ring/50 data-[state=open]:ring-[3px]",
          !current && "text-muted-foreground",
          className,
        )}
        disabled={disabled}
        onClick={() => !useDialog || setOpen(true)}
      >
        <p className="truncate">
          {current ? getSelectedLabel(items, current) : `选择${formTitle}`}
        </p>
        <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
      </Button>
    </FormControl>
  );

  const commandContent = (
    <Command>
      <CommandInput placeholder={`查找${formTitle}`} className="h-9" />
      <CommandList>
        <CommandEmpty>未找到匹配的{formTitle}</CommandEmpty>
        <CommandGroup>
          <div>
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
        </CommandGroup>
      </CommandList>
    </Command>
  );

  return useDialog ? (
    <>
      {triggerButton}
      <CommandDialog open={open} onOpenChange={setOpen}>
        {commandContent}
      </CommandDialog>
    </>
  ) : (
    <Popover modal open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{triggerButton}</PopoverTrigger>
      <PopoverContent
        className="z-50 p-0"
        style={{
          width: "var(--radix-popover-trigger-width)",
          maxHeight: "var(--radix-popover-content-available-height)",
        }}
      >
        {commandContent}
      </PopoverContent>
    </Popover>
  );
}

export default Combobox;
