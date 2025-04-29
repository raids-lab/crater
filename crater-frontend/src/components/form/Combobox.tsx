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
import { useMemo, useState } from "react";
import { TagFilter, useTagFilter } from "./ImageFormField";
import { Badge } from "@/components/ui/badge";
export interface ComboboxItem<T> {
  label: string;
  value: string;
  selectedLabel?: string;
  tags?: string[];
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
  tags?: string[]; // Optional predefined tags
  tagFilter?: React.ReactNode; // Optional custom tag filter component
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
  tags,
  tagFilter,
}: ComboboxProps<T>) {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  // Use the tag filter hook
  const { allTags, selectedTags, toggleTag, filterItemsByTags } = useTagFilter(
    items,
    tags,
  );

  // Filter items based on search query and selected tags
  const filteredItems = useMemo(() => {
    // First filter by tags
    const tagFilteredItems = filterItemsByTags(items);

    // Then filter by search query
    return tagFilteredItems.filter((item) => {
      return (
        searchQuery === "" ||
        item.label.toLowerCase().includes(searchQuery.toLowerCase())
      );
    });
  }, [items, searchQuery, filterItemsByTags]);

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
      <CommandInput
        placeholder={`查找${formTitle}`}
        className="h-9"
        onValueChange={setSearchQuery}
      />

      {tagFilter ||
        (allTags.length > 0 && (
          <TagFilter
            tags={allTags}
            selectedTags={selectedTags}
            onTagToggle={toggleTag}
          />
        ))}

      <CommandList>
        <CommandEmpty>未找到匹配的{formTitle}</CommandEmpty>
        <CommandGroup>
          <div>
            {filteredItems.map((item) => (
              <CommandItem
                value={item.label}
                key={item.value}
                onSelect={() => {
                  handleSelect(item.value);
                  setOpen(false);
                }}
                className="flex w-full flex-row items-center justify-between"
              >
                <div className="flex flex-col">
                  <div>{renderLabel ? renderLabel(item) : item.label}</div>
                  {item.tags && item.tags.length > 0 && (
                    <div className="mt-1 flex gap-1">
                      {item.tags.map((tag) => (
                        <Badge
                          key={tag}
                          variant="secondary"
                          className="text-xs"
                        >
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  )}
                </div>
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
