import { Button } from "@/components/ui/button";
import DataList from "@/pages/Portal/Data/DataList";
import { BookOpenIcon } from "lucide-react"; //, CirclePlusIcon
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
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
import { AxiosResponse } from "axios";
interface DatesetTableProps {
  apiGetDataset: () => Promise<AxiosResponse<IResponse<Dataset[]>>>;
}
export function ModelTable({ apiGetDataset }: DatesetTableProps) {
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
          ?.filter((dataset) => dataset.type === "model")
          .map((dataset) => ({
            id: dataset.id,
            name: dataset.name,
            desc: dataset.describe,
            tag: dataset.extra.tag || [], // Add appropriate tags if available
            createdAt: dataset.createdAt,
            owner: dataset.IUserAttributes, // Adjust to match IUserAttributes structure
            username: dataset.username,
          })) || []
      }
      title="模型"
      actionArea={
        <div className="flex flex-row gap-3">
          <Button variant="secondary">
            <BookOpenIcon />
            模型文档
          </Button>
          {/*<Button>
            <CirclePlusIcon />
            导入模型
          </Button>*/}
          <Sheet open={openSheet} onOpenChange={setOpenSheet}>
            <SheetTrigger asChild>
              <Button className="h-8 min-w-fit">创建模型</Button>
            </SheetTrigger>
            <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
              <SheetHeader>
                <SheetTitle>创建新模型</SheetTitle>
                <SheetDescription>创建一个新的文件模型集</SheetDescription>
              </SheetHeader>
              <Separator className="mt-4" />
              <DatasetCreateForm
                closeSheet={() => setOpenSheet(false)}
                type="model"
              />
            </SheetContent>
          </Sheet>
        </div>
      }
    />
  );
}
/*const Model = () => {
  
};

export default Model;*/
