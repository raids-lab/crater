import { useState } from "react";
import { PaginationState } from "@tanstack/react-table";
import { useLocalStorage } from "usehooks-ts";

export function usePaginationWithStorage(tableKey: string = "default-table") {
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useLocalStorage(`${tableKey}-page-size`, 10);

  const pagination: PaginationState = {
    pageIndex,
    pageSize,
  };

  const setPagination = (
    updater: PaginationState | ((state: PaginationState) => PaginationState),
  ) => {
    const newState =
      typeof updater === "function" ? updater(pagination) : updater;
    setPageIndex(newState.pageIndex);
    setPageSize(newState.pageSize);
  };

  return [pagination, setPagination] as const;
}
