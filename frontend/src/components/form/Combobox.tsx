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
} from "@/components/ui/command";
import { Check, ChevronsUpDown } from "lucide-react";
// import { ScrollArea } from "@/components/ui/scroll-area";
import { useState } from "react";

export interface ComboboxItem {
  label: string;
  value: string;
}

type ComboboxProps = React.HTMLAttributes<HTMLDivElement> & {
  formTitle: string;
  items: ComboboxItem[];
  current: string;
  handleSelect: (value: string) => void;
  className?: string;
};

const Combobox = ({
  formTitle,
  items,
  current,
  handleSelect,
  className,
}: ComboboxProps) => {
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <FormControl>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className={cn(
              "w-full justify-between text-ellipsis whitespace-nowrap font-normal",
              !current && "text-muted-foreground",
              className,
            )}
          >
            {current
              ? items.find((dataset) => dataset.value === current)?.label
              : `选择${formTitle}`}
            <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </FormControl>
      </PopoverTrigger>
      <PopoverContent
        className="p-0"
        style={{
          width: "var(--radix-popover-trigger-width)",
          maxHeight: "var(--radix-popover-content-available-height)",
        }}
      >
        <Command>
          <CommandInput placeholder={`查找${formTitle}`} className="h-9" />
          <CommandEmpty>未找到匹配的{formTitle}</CommandEmpty>
          {/* <ScrollArea className="h-40"> */}
          <CommandGroup>
            {items.map((image) => (
              <CommandItem
                value={image.label}
                key={image.value}
                onSelect={() => {
                  handleSelect(image.value);
                  setOpen(false);
                }}
                className="flex w-full flex-row items-center justify-between"
              >
                {image.label}
                <Check
                  className={cn(
                    "ml-auto h-4 w-4",
                    image.value === current ? "opacity-100" : "opacity-0",
                  )}
                />
              </CommandItem>
            ))}
          </CommandGroup>
          {/* </ScrollArea> */}
        </Command>
      </PopoverContent>
    </Popover>
  );
};

export default Combobox;
