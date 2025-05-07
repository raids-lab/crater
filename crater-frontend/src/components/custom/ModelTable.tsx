import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { IDataset } from "@/services/api/dataset";
import DocsButton from "@/components/button/DocsButton";
import { DataCreateForm } from "@/pages/Portal/Data/CreateForm";
import { IResponse } from "@/services/types";
import DataList from "@/pages/Portal/Data/DataList";
import { AxiosResponse } from "axios";
import SandwichSheet from "@/components/sheet/SandwichSheet";
import { PlusIcon } from "lucide-react";

interface DatesetTableProps {
  sourceType?: "dataset" | "model" | "sharefile";
  apiGetDataset: () => Promise<AxiosResponse<IResponse<IDataset[]>>>;
}

export function ModelTable({ apiGetDataset, sourceType }: DatesetTableProps) {
  const data = useQuery({
    queryKey: ["data", "mydataset"],
    queryFn: () => apiGetDataset(),
    select: (res) => res.data.data,
  });
  const sourcetitle = sourceType === "model" ? "模型" : "共享文件";
  const [openSheet, setOpenSheet] = useState(false);

  return (
    <DataList
      items={
        data.data
          ?.filter((dataset) => dataset.type === sourceType)
          .map((dataset) => ({
            id: dataset.id,
            name: dataset.name,
            desc: dataset.describe,
            tag: dataset.extra.tag || [],
            createdAt: dataset.createdAt,
            owner: dataset.userInfo,
          })) || []
      }
      title={sourcetitle}
      actionArea={
        <div className="flex flex-row gap-3">
          {sourceType !== "sharefile" && (
            <DocsButton title={`${sourcetitle}文档`} url="file/model" />
          )}
          <SandwichSheet
            isOpen={openSheet}
            onOpenChange={setOpenSheet}
            title={`创建${sourcetitle}`}
            description={`创建一个新的${sourcetitle}`}
            trigger={
              <Button className="min-w-fit">
                <PlusIcon />
                添加{sourcetitle}
              </Button>
            }
            className="sm:max-w-3xl"
          >
            <DataCreateForm
              closeSheet={() => setOpenSheet(false)}
              type={sourceType}
            />
          </SandwichSheet>
        </div>
      }
    />
  );
}
