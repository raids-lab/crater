import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import React, { useEffect, type FC } from "react";
import { Card } from "@/components/ui/card";
import { useParams } from "react-router-dom";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import {
  apiUserDeleteKaniko,
  apiUserGetKaniko,
  KanikoInfoResponse,
} from "@/services/api/imagepack";
import {
  Clock,
  Hash,
  Link2,
  LogsIcon,
  CodeIcon,
  ActivityIcon,
  Trash2Icon,
} from "lucide-react";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { DetailPage } from "@/components/layout/DetailPage";
import PageTitle from "@/components/layout/PageTitle";
import { shortenImageName } from "@/utils/formatter";
import { CodeContent } from "@/components/codeblock/ConfigDialog";
import DetailPageLog from "@/components/codeblock/DetailPageLog";
import ImagePhaseBadge from "@/components/badge/ImagePhaseBadge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui-custom/alert-dialog";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

type KanikoCard = React.ComponentProps<typeof Card> & {
  kanikoInfo?: KanikoInfoResponse;
};

function KanikoInfo({ kanikoInfo: kanikoInfo }: KanikoCard) {
  const queryClient = useQueryClient();

  const { mutate: userDeleteKaniko } = useMutation({
    mutationFn: (id: number) => apiUserDeleteKaniko(id),
    onSuccess: async () => {
      await new Promise((resolve) => setTimeout(resolve, 500)).then(() =>
        queryClient.invalidateQueries({ queryKey: ["imagelink", "list"] }),
      );
      toast.success("镜像已删除");
    },
  });

  if (!kanikoInfo) {
    return null;
  }

  return (
    <>
      <DetailPage
        header={
          <PageTitle
            title={kanikoInfo.description}
            // description={shortestImageName(kanikoInfo.imageLink)}
          >
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <div>
                  <Button variant="destructive" title="删除镜像">
                    <Trash2Icon className="size-4" />
                    删除镜像
                  </Button>
                </div>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除镜像构建任务</AlertDialogTitle>
                  <AlertDialogDescription>
                    镜像 {shortenImageName(kanikoInfo.imageLink)}{" "}
                    将被删除，请确认镜像已不再需要。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => {
                      userDeleteKaniko(kanikoInfo.ID);
                    }}
                  >
                    删除
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </PageTitle>
        }
        info={[
          {
            icon: Hash,
            title: "ID",
            value: kanikoInfo.ID,
          },
          {
            icon: Link2,
            title: "URL",
            value: (
              <span className="font-mono">
                {shortenImageName(kanikoInfo?.imageLink)}
              </span>
            ),
            className: "col-span-2",
          },
          {
            icon: Clock,
            title: "创建于",
            value: <TimeDistance date={kanikoInfo.createdAt} />,
          },
          {
            icon: ActivityIcon,
            title: "状态",
            value: <ImagePhaseBadge status={kanikoInfo.status} />,
          },
        ]}
        tabs={[
          {
            key: "info",
            icon: LogsIcon,
            label: "构建日志",
            children: (
              <DetailPageLog
                namespacedName={{
                  namespace: kanikoInfo?.podNameSpace ?? "",
                  name: kanikoInfo?.podName ?? "",
                }}
              />
            ),
            hidden: kanikoInfo?.podName === "",
          },
          {
            key: "dockerfile",
            icon: CodeIcon,
            label: "Dockerfile",
            children: (
              <CodeContent
                data={kanikoInfo.dockerfile}
                language={"dockerfile"}
              />
            ),
          },
        ]}
      />
    </>
  );
}

export const KanikoDetail: FC = () => {
  const { id: kanikoID } = useParams();
  const { data: kanikoInfo } = useQuery({
    queryKey: ["imagepack", "get", kanikoID],
    queryFn: () => apiUserGetKaniko(`${kanikoID}`),
    select: (res) => res.data.data,
    enabled: !!kanikoID,
  });

  const setBreadcrumb = useBreadcrumb(); // 修改 BreadCrumb
  useEffect(() => {
    setBreadcrumb([{ title: "构建详情" }]);
  }, [setBreadcrumb]);

  return <KanikoInfo kanikoInfo={kanikoInfo} />;
};

export default KanikoDetail;
