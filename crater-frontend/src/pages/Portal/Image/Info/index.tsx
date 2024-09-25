import { useQuery } from "@tanstack/react-query";
import { useEffect, type FC } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useParams } from "react-router-dom";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import {
  apiUserImageCreateGet,
  apiUserImageCreateLog,
  imagepackStatuses,
  imagepackTaskType,
} from "@/services/api/imagepack";
import { TableDate } from "@/components/custom/TableDate";
import { cn } from "@/lib/utils";

type ImageCreateInfoCard = React.ComponentProps<typeof Card> & {
  imagecreateinfo?: {
    ID: number;
    imagelink: string;
    status: string;
    createdAt: string;
    nametag: string;
    creatername: string;
    imagetype: number;
    description: string;
    alias: string;
    tasktype: number;
    params: {
      Convs: number;
      Activations: number;
      Denses: number;
      Others: number;
      GFLOPs: number;
      BatchSize: number;
      Params: number;
      ModelSize: number;
    };
  };
};

// CardDemo 组件
function ImageCreateInfo({
  className,
  imagecreateinfo,
  ...props
}: ImageCreateInfoCard) {
  const status = imagepackStatuses.find(
    (status) => status.value === imagecreateinfo?.status,
  );
  const tasktype = imagepackTaskType.find(
    (tasktype) => tasktype.value === imagecreateinfo?.tasktype,
  );
  return (
    <Card className={className} {...props}>
      <CardHeader>
        <CardTitle>镜像基础信息</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4">
        <div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">ID:</p>
            <p className="ml-2 text-sm font-medium">{imagecreateinfo?.ID}</p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">镜像链接:</p>
            <p className="ml-2 text-sm font-medium">
              {imagecreateinfo?.imagelink}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">用户:</p>
            <p className="ml-2 text-sm font-medium">
              {imagecreateinfo?.creatername}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">状态:</p>
            {/* <p className="ml-2 text-sm font-medium">{imagecreateinfo?.status}</p> */}
            <div className="flex flex-row items-center justify-start">
              <div
                className={cn("flex h-3 w-3 rounded-full", {
                  "bg-purple-500 hover:bg-purple-400":
                    status?.value === "Initial",
                  "bg-slate-500 hover:bg-slate-400":
                    status?.value === "Pending",
                  "bg-sky-500 hover:bg-sky-400": status?.value === "Running",
                  "bg-red-500 hover:bg-red-400": status?.value === "Failed",
                  "bg-emerald-500 hover:bg-emerald-400":
                    status?.value === "Finished",
                })}
              ></div>
              <div className="ml-1.5">{status?.label}</div>
            </div>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">创建时间:</p>
            {/* <p className="ml-2 text-sm font-medium">{imagecreateinfo?.createdAt}</p> */}
            <p className="ml-2 text-sm font-medium">
              {
                <TableDate
                  date={
                    imagecreateinfo?.createdAt ?? "2024-05-16T11:02:34.293433Z"
                  }
                ></TableDate>
              }
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">别名:</p>
            <p className="ml-2 text-sm font-medium">{imagecreateinfo?.alias}</p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">描述:</p>
            <p className="ml-2 text-sm font-medium">
              {imagecreateinfo?.description}
            </p>
          </div>
          <div className="mb-2 flex items-center pb-2 last:mb-0 last:pb-0">
            <p className="text-xs">任务类型:</p>
            <p className="ml-2 text-sm font-medium">{tasktype?.label}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export function ImageParamsInfo({
  className,
  imagecreateinfo,
  ...props
}: ImageCreateInfoCard) {
  return (
    <Card className={className} {...props}>
      <CardHeader>
        <CardTitle>镜像参数信息</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4">
        <div>Convs: {imagecreateinfo?.params.Convs}</div>
        <div>Activations: {imagecreateinfo?.params.Activations}</div>
        <div>Denses: {imagecreateinfo?.params.Denses}</div>
        <div>Others: {imagecreateinfo?.params.Others}</div>
        <div>GFLOPs: {imagecreateinfo?.params.GFLOPs.toFixed(2)}</div>
        <div>BatchSize: {imagecreateinfo?.params.BatchSize}</div>
        <div>Params: {imagecreateinfo?.params.Params}</div>
        <div>ModelSize: {imagecreateinfo?.params.ModelSize.toFixed(2)}</div>
      </CardContent>
    </Card>
  );
}

export const ImageCreateDetail: FC = () => {
  const { id: imageID } = useParams();
  const { data: imagecreateinfo } = useQuery({
    queryKey: ["imagepack", "get", imageID],
    queryFn: () => apiUserImageCreateGet(`${imageID}`),
    select: (res) => res.data.data,
    enabled: !!imageID,
  });

  const { data: imageloginfo } = useQuery({
    queryKey: ["imagepack", "log", imageID],
    queryFn: () => apiUserImageCreateLog(`${imageID}`),
    select: (res) => res.data.data,
    enabled: !!imageID,
  });

  const setBreadcrumb = useBreadcrumb();

  // 修改 BreadCrumb
  useEffect(() => {
    setBreadcrumb([{ title: imageID ?? "" }]);
  }, [setBreadcrumb, imageID]);

  return (
    <>
      <div className="grid-auto-flow: dense; grid auto-rows-max items-start gap-4 md:gap-8 lg:col-span-2">
        <div className="grid-auto-rows: 1fr; grid gap-4 sm:grid-cols-2 md:grid-cols-4 lg:grid-cols-2 xl:grid-cols-4">
          <ImageCreateInfo
            className="sm:col-span-2"
            imagecreateinfo={imagecreateinfo}
          />
          <ImageParamsInfo
            className="sm:col-span-2"
            imagecreateinfo={imagecreateinfo}
          />
        </div>
      </div>
      <div className="col-span-3 flex-none">
        {imageloginfo && (
          <Card className="col-span-3 bg-gray-100 text-muted-foreground dark:border dark:bg-transparent">
            <CardHeader className="py-3"></CardHeader>
            <CardContent>
              <pre className="whitespace-pre-wrap text-sm">
                {imageloginfo.content}
              </pre>
            </CardContent>
          </Card>
        )}
      </div>
    </>
  );
};

export default ImageCreateDetail;
