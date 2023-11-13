import { Cross2Icon } from "@radix-ui/react-icons";
import { Table } from "@tanstack/react-table";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DataTableViewOptions } from "./DataTableViewOptions";

import {
  DataTableFacetedFilter,
  DataTableFacetedFilterOption,
} from "./DataTableFacetedFilter";

export interface DataTableToolbarConfig {
  filterInput: {
    placeholder: string;
    key: string;
  };
  filterOptions: {
    key: string;
    title: string;
    option: DataTableFacetedFilterOption[];
  }[];
}

interface DataTableToolbarProps<TData> {
  table: Table<TData>;
  config: DataTableToolbarConfig;
}

export function DataTableToolbar<TData>({
  table,
  config: { filterInput, filterOptions },
}: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters.length > 0;

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-1 items-center space-x-2">
        <Input
          placeholder={filterInput.placeholder}
          value={
            (table.getColumn(filterInput.key)?.getFilterValue() as string) ?? ""
          }
          onChange={(event) =>
            table.getColumn(filterInput.key)?.setFilterValue(event.target.value)
          }
          className="h-8 w-[150px] lg:w-[250px]"
        />
        {filterOptions.map(
          (filterOption) =>
            table.getColumn(filterOption.key) && (
              <DataTableFacetedFilter
                key={filterOption.key}
                column={table.getColumn(filterOption.key)}
                title={filterOption.title}
                options={filterOption.option}
              />
            ),
        )}
        {isFiltered && (
          <Button
            variant="outline"
            size="icon"
            title="Clear filters"
            type="button"
            onClick={() => table.resetColumnFilters()}
            className="h-8 w-8 border-dashed"
          >
            <Cross2Icon className="h-4 w-4" />
          </Button>
        )}
      </div>
      <DataTableViewOptions table={table} />
    </div>
  );
}
