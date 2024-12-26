import { Table } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DataTableViewOptions } from "./DataTableViewOptions";
import {
  DataTableFacetedFilter,
  DataTableFacetedFilterOption,
} from "./DataTableFacetedFilter";
import { SearchIcon, XIcon } from "lucide-react";

export interface DataTableToolbarConfig {
  filterInput: {
    placeholder: string;
    key: string;
  };
  filterOptions: readonly {
    key: string;
    title: string;
    option: DataTableFacetedFilterOption[];
    defaultValues?: string[];
  }[];
  getHeader: (key: string) => string;
}

interface DataTableToolbarProps<TData>
  extends React.HTMLAttributes<HTMLDivElement> {
  table: Table<TData>;
  config: DataTableToolbarConfig;
  isLoading: boolean;
}

export function DataTableToolbar<TData>({
  table,
  config: { filterInput, filterOptions, getHeader },
  isLoading,
  children,
}: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters.length > 0;

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-row items-center space-x-2">
        {children}
        <div className="relative ml-auto h-9 flex-1 md:grow-0">
          <SearchIcon className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
          <Input
            placeholder={filterInput.placeholder}
            value={
              (table.getColumn(filterInput.key)?.getFilterValue() as string) ??
              ""
            }
            onChange={(event) =>
              table
                .getColumn(filterInput.key)
                ?.setFilterValue(event.target.value)
            }
            className="h-9 w-[150px] bg-background pl-8 lg:w-[250px]"
          />
        </div>
        {filterOptions.map(
          (filterOption) =>
            table.getColumn(filterOption.key) && (
              <DataTableFacetedFilter
                key={filterOption.key}
                column={table.getColumn(filterOption.key)}
                title={filterOption.title}
                options={filterOption.option}
                defaultValues={filterOption.defaultValues}
              />
            ),
        )}
        {isFiltered && !isLoading && (
          <Button
            variant="outline"
            size="icon"
            title="Clear filters"
            type="button"
            onClick={() => table.resetColumnFilters()}
            className="size-9 border-dashed"
          >
            <XIcon className="size-4" />
          </Button>
        )}
      </div>
      <DataTableViewOptions table={table} getHeader={getHeader} />
    </div>
  );
}
