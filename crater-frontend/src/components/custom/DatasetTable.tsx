//import type { FC } from "react";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Dataset } from "@/services/api/dataset";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Separator } from "@radix-ui/react-dropdown-menu";
import { DatasetCreateForm } from "@/pages/Portal/Data/CreateForm";

import { IResponse } from "@/services/types";
import DataList from "@/pages/Portal/Data/DataList";
import { AxiosResponse } from "axios";
interface DatesetTableProps {
  apiGetDataset: () => Promise<AxiosResponse<IResponse<Dataset[]>>>;
}
export function DatasetTable({ apiGetDataset }: DatesetTableProps) {
  const data = useQuery({
    queryKey: ["data", "mydataset"],
    queryFn: () => apiGetDataset(),
    select: (res) => res.data.data,
  });

  const [openSheet, setOpenSheet] = useState(false);

  return (
    <DataList
      items={
        data.data
          ?.filter((dataset) => dataset.type === "dataset")
          .map((dataset) => ({
            id: dataset.id,
            name: dataset.name,
            desc: dataset.describe,
            tags: dataset.tags || [], // Add appropriate tags if available
            owner: dataset.IUserAttributes, // Adjust to match IUserAttributes structure
            username: dataset.username,
          })) || []
      }
      title="数据集"
      actionArea={
        <Sheet open={openSheet} onOpenChange={setOpenSheet}>
          <SheetTrigger asChild>
            <Button className="h-8 min-w-fit">创建数据集</Button>
          </SheetTrigger>
          <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
            <SheetHeader>
              <SheetTitle>创建数据集</SheetTitle>
              <SheetDescription>创建一个新的文件数据集</SheetDescription>
            </SheetHeader>
            <Separator className="mt-4" />
            <DatasetCreateForm closeSheet={() => setOpenSheet(false)} />
          </SheetContent>
        </Sheet>
      }
    />
  );
}
