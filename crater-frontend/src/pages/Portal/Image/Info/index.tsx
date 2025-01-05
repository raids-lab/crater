import { useQuery } from "@tanstack/react-query";
import React, { useEffect, type FC } from "react";
import { Card } from "@/components/ui/card";
import { useParams } from "react-router-dom";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { apiUserGetKaniko, imagepackStatuses } from "@/services/api/imagepack";
import { cn } from "@/lib/utils";
import {
  Clock,
  Hash,
  Link2,
  LogsIcon,
  CodeIcon,
  ActivityIcon,
} from "lucide-react";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { DetailPage } from "@/components/layout/DetailPage";
import PageTitle from "@/components/layout/PageTitle";
import { shortenImageName } from "@/utils/formatter";
import { CodeContent } from "@/components/codeblock/ConfigDialog";
import DetailPageLog from "@/components/codeblock/DetailPageIog";

type KanikoCard = React.ComponentProps<typeof Card> & {
  kanikoInfo?: {
    ID: number;
    imageLink: string;
    status: string;
    createdAt: string;
    dockerfile: string;
    description: string;
    podName: string;
    podNameSpace: string;
  };
};

function KanikoInfo({ kanikoInfo: kanikoInfo }: KanikoCard) {
  const status = imagepackStatuses.find(
    (status) => status.value === kanikoInfo?.status,
  );

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
          />
        }
        info={[
          {
            icon: Hash,
            title: "ID",
            value: kanikoInfo.ID,
          },
          {
            icon: Clock,
            title: "创建于",
            value: <TimeDistance date={kanikoInfo.createdAt} />,
          },
          {
            icon: ActivityIcon,
            title: "状态",
            value: (
              <div className="flex items-center gap-2">
                <div
                  className={cn("h-2.5 w-2.5 rounded-full", {
                    "bg-purple-500": status?.value === "Initial",
                    "bg-slate-500": status?.value === "Pending",
                    "bg-sky-500": status?.value === "Running",
                    "bg-red-500": status?.value === "Failed",
                    "bg-emerald-500": status?.value === "Finished",
                  })}
                />
                <span className="text-sm font-medium">{status?.label}</span>
              </div>
            ),
          },
          {
            icon: Link2,
            title: "URL",
            value: shortenImageName(kanikoInfo?.imageLink),
            className: "col-span-3",
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
