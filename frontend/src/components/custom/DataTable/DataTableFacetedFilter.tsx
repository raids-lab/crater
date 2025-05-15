// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
import * as React from "react";
import { CheckIcon } from "@radix-ui/react-icons";
import { Column } from "@tanstack/react-table";
import { useMemo, useCallback, useEffect } from "react";

import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  // CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { ListFilter } from "lucide-react";

export interface DataTableFacetedFilterOption {
  label: string;
  value: string;
  icon?: React.ComponentType<{ className?: string }>;
}

interface DataTableFacetedFilterProps<TData, TValue> {
  column?: Column<TData, TValue>;
  title?: string;
  options: DataTableFacetedFilterOption[];
  defaultValues?: string[];
}

function DataTableFacetedFilterComponent<TData, TValue>({
  column,
  title,
  options,
  defaultValues,
}: DataTableFacetedFilterProps<TData, TValue>) {
  const { t } = useTranslation();
  const facets = column?.getFacetedUniqueValues();
  const selectedValues = useMemo(
    () => new Set(column?.getFilterValue() as string[]),
    [column],
  );

  // 使用 useCallback 记忆清除过滤器的回调函数
  const handleClearFilter = useCallback(() => {
    column?.setFilterValue(undefined);
  }, [column]);

  // 计算过滤后的选项列表
  const filteredOptions = useMemo(
    () => options.filter((option) => 0 < (facets?.get(option.value) || 0)),
    [options, facets],
  );

  // 计算选中的选项标签
  const selectedBadges = useMemo(() => {
    if (selectedValues.size > 2) {
      return (
        <Badge variant="secondary" className="rounded-sm px-1 font-normal">
          {t("dataTableFacetedFilter.selected", {
            count: selectedValues.size,
          })}
        </Badge>
      );
    }

    return options
      .filter((option) => selectedValues.has(option.value))
      .map((option) => (
        <Badge
          variant="secondary"
          key={option.value}
          className="rounded-sm px-1 font-normal"
        >
          {option.label}
        </Badge>
      ));
  }, [selectedValues, options, t]);

  // 设置默认过滤选项
  useEffect(() => {
    if (defaultValues) {
      column?.setFilterValue(defaultValues);
    }
  }, [defaultValues, column]);

  // 记忆选择处理函数
  const handleOptionSelect = useCallback(
    (optionValue: string) => {
      return () => {
        const newSelectedValues = new Set(selectedValues);
        if (newSelectedValues.has(optionValue)) {
          newSelectedValues.delete(optionValue);
        } else {
          newSelectedValues.add(optionValue);
        }
        const filterValues = Array.from(newSelectedValues);
        column?.setFilterValue(filterValues.length ? filterValues : undefined);
      };
    },
    [selectedValues, column],
  );

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className="h-9 border-dashed"
          disabled={facets?.size === 0}
        >
          <ListFilter className="size-4" />
          <span className="text-xs">{title}</span>
          {selectedValues?.size > 0 && (
            <>
              <Separator orientation="vertical" className="h-4" />
              <Badge
                variant="secondary"
                className="rounded-sm px-1 font-normal lg:hidden"
              >
                {selectedValues.size}
              </Badge>
              <div className="hidden space-x-1 lg:flex">{selectedBadges}</div>
            </>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0" align="start">
        <Command shouldFilter={false}>
          {/* <CommandInput placeholder={title} /> */}
          <CommandList>
            <CommandEmpty>{t("dataTableFacetedFilter.noResults")}</CommandEmpty>
            <CommandGroup>
              {filteredOptions.map((option) => {
                const isSelected = selectedValues.has(option.value);
                return (
                  <CommandItem
                    key={option.value}
                    onSelect={handleOptionSelect(option.value)}
                  >
                    <div
                      className={cn(
                        "border-primary flex size-4 items-center justify-center rounded-sm border",
                        isSelected
                          ? "bg-primary text-primary-foreground"
                          : "opacity-50 [&_svg]:invisible",
                      )}
                    >
                      <CheckIcon
                        className={cn("text-primary-foreground size-4")}
                      />
                    </div>
                    {option.icon && (
                      <option.icon className="text-muted-foreground mr-2 size-4" />
                    )}
                    <span>{option.label}</span>
                    {facets?.get(option.value) && (
                      <span className="ml-auto flex size-4 items-center justify-center font-mono text-xs">
                        {facets.get(option.value)}
                      </span>
                    )}
                  </CommandItem>
                );
              })}
            </CommandGroup>
            {selectedValues.size > 0 && (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem
                    onSelect={handleClearFilter}
                    className="justify-center text-center"
                  >
                    {t("dataTableFacetedFilter.clearFilter")}
                  </CommandItem>
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

export const DataTableFacetedFilter = React.memo(
  DataTableFacetedFilterComponent,
) as typeof DataTableFacetedFilterComponent;
