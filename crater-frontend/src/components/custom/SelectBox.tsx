// Copied from https://github.com/shadcn-ui/ui/issues/927#issuecomment-2272458201
import { CheckIcon, Cross2Icon } from "@radix-ui/react-icons";
import React from "react";

import { cn } from "@/lib/utils";

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { DialogOverlay } from "@radix-ui/react-dialog";
import { ChevronsUpDown, XIcon } from "lucide-react";

interface Option {
  value: string;
  label: string;
  labelNote?: string;
}

interface SelectBoxProps {
  options: Option[];
  value?: string[];
  onChange?: (values: string[]) => void;
  placeholder?: string;
  inputPlaceholder?: string;
  emptyPlaceholder?: string;
  className?: string;
}

const SelectBox = React.forwardRef<HTMLInputElement, SelectBoxProps>(
  (
    {
      inputPlaceholder,
      emptyPlaceholder,
      placeholder,
      className,
      options,
      value,
      onChange,
    },
    ref,
  ) => {
    const [searchTerm, setSearchTerm] = React.useState<string>("");
    const [isOpen, setIsOpen] = React.useState(false);

    const handleSelect = (selectedValue: string) => {
      const newValue =
        value?.includes(selectedValue) && Array.isArray(value)
          ? value.filter((v) => v !== selectedValue)
          : [...(value ?? []), selectedValue];
      onChange?.(newValue);
    };

    const handleClear = () => {
      onChange?.([]);
    };

    return (
      <Popover open={isOpen} onOpenChange={setIsOpen}>
        <PopoverTrigger asChild>
          <Button
            variant={"outline"}
            className={cn(
              "flex h-full min-h-[36px] w-full cursor-pointer items-center justify-between whitespace-nowrap px-3 py-1 font-normal",
              "data-[state=open]:outline-none data-[state=open]:ring-1 data-[state=open]:ring-ring",
              className,
            )}
          >
            <div
              className={cn(
                "items-center gap-1 overflow-hidden text-sm",
                "flex flex-grow flex-wrap",
              )}
            >
              {value && value.length > 0 ? (
                options
                  .filter((option) => value.includes(option.value))
                  ?.map((option) => (
                    <span
                      key={option.value}
                      className="inline-flex items-center gap-1 rounded-md border bg-secondary py-0.5 pl-2 pr-1 text-xs font-medium text-secondary-foreground transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                    >
                      <span>{option.label}</span>
                      <span
                        onClick={(e) => {
                          e.preventDefault();
                          handleSelect(option.value);
                        }}
                        className="flex items-center rounded-sm px-[1px] text-muted-foreground/60 hover:bg-accent hover:text-muted-foreground"
                      >
                        <Cross2Icon />
                      </span>
                    </span>
                  ))
              ) : (
                <span className="mr-auto text-muted-foreground">
                  {placeholder}
                </span>
              )}
            </div>
            <div className="flex items-center self-stretch pl-1 text-muted-foreground hover:text-foreground [&>div]:flex [&>div]:items-center [&>div]:self-stretch">
              {value && value.length > 0 ? (
                <div
                  onClick={(e) => {
                    e.preventDefault();
                    handleClear();
                  }}
                >
                  <XIcon className="size-4" />
                </div>
              ) : (
                <div>
                  <ChevronsUpDown className="h-4 w-4 shrink-0 opacity-50" />
                </div>
              )}
            </div>
          </Button>
        </PopoverTrigger>
        <DialogOverlay>
          <PopoverContent
            className="w-[var(--radix-popover-trigger-width)] p-0"
            align="start"
          >
            <Command>
              <div className="relative">
                <CommandInput
                  value={searchTerm}
                  onValueChange={(e) => setSearchTerm(e)}
                  ref={ref}
                  placeholder={inputPlaceholder ?? "Search..."}
                  className="h-9"
                />
                {searchTerm && (
                  <div
                    className="absolute inset-y-0 right-0 flex cursor-pointer items-center pr-3 text-muted-foreground hover:text-foreground"
                    onClick={() => setSearchTerm("")}
                  >
                    <Cross2Icon className="size-4" />
                  </div>
                )}
              </div>
              <CommandList>
                <CommandEmpty>
                  {emptyPlaceholder ?? "No results found."}
                </CommandEmpty>
                <CommandGroup>
                  <ScrollArea>
                    <div className="max-h-64">
                      {options?.map((option) => {
                        const isSelected =
                          Array.isArray(value) && value.includes(option.value);
                        return (
                          <CommandItem
                            key={option.value}
                            // value={option.value}
                            onSelect={() => handleSelect(option.value)}
                          >
                            <div
                              className={cn(
                                "mr-2 flex h-4 w-4 items-center justify-center rounded-sm border border-primary",
                                isSelected
                                  ? "bg-primary text-primary-foreground"
                                  : "opacity-50 [&_svg]:invisible",
                              )}
                            >
                              <CheckIcon />
                            </div>
                            <span>
                              {option.label}
                              <span className="ml-2 text-muted-foreground">
                                {"@"}
                                {option.labelNote}
                              </span>
                            </span>
                          </CommandItem>
                        );
                      })}
                    </div>
                  </ScrollArea>
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </DialogOverlay>
      </Popover>
    );
  },
);

SelectBox.displayName = "SelectBox";

export default SelectBox;
