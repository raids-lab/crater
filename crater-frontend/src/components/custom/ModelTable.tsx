import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { IDataset } from "@/services/api/dataset";
import DocsButton from "@/components/button/DocsButton";
import { DatasetCreateForm } from "@/pages/Portal/Data/CreateForm";
import { IResponse } from "@/services/types";
import DataList from "@/pages/Portal/Data/DataList";
import { AxiosResponse } from "axios";
import SandwichSheet from "@/components/sheet/SandwichSheet";

interface DatesetTableProps {
  apiGetDataset: () => Promise<AxiosResponse<IResponse<IDataset[]>>>;
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
            tag: dataset.extra.tag || [],
            createdAt: dataset.createdAt,
            owner: dataset.userInfo,
          })) || []
      }
      title="模型"
      actionArea={
        <div className="flex flex-row gap-3">
          <DocsButton title="模型文档" url="file/model" />
          <SandwichSheet
            isOpen={openSheet}
            onOpenChange={setOpenSheet}
            title="创建模型"
            description="创建一个新的文件模型集"
            trigger={<Button className="h-8 min-w-fit">创建模型</Button>}
            className="sm:max-w-3xl"
          >
            <div className="pt-1">
              <DatasetCreateForm
                closeSheet={() => setOpenSheet(false)}
                type="model"
              />
            </div>
          </SandwichSheet>
        </div>
      }
    />
  );
}
