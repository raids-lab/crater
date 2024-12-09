import { useQuery } from "@tanstack/react-query";
import React, { useEffect, useState, type FC } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useParams } from "react-router-dom";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { apiUserGetKaniko, imagepackStatuses } from "@/services/api/imagepack";
import { cn } from "@/lib/utils";
import {
  Clock,
  Hash,
  Link2,
  FileText,
  Terminal,
  Copy,
  CheckCircle,
  Info,
} from "lucide-react";
import { ScrollArea } from "@radix-ui/react-scroll-area";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { NamespacedName } from "@/components/codeblock/PodContainerDialog";
import LogDialog from "@/components/codeblock/LogDialog";
import { TimeDistance } from "@/components/custom/TimeDistance";

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

interface InfoItemProps {
  icon: React.ReactNode;
  label: string;
  value?: string | number;
  className?: string;
  copyable?: boolean;
  onCopy?: () => void;
}

function InfoItem({
  icon,
  label,
  value,
  className,
  copyable,
  onCopy,
}: InfoItemProps) {
  return (
    <div className="group flex items-start gap-3">
      <span className="mt-0.5 text-muted-foreground">{icon}</span>
      <div>
        <span className="text-sm font-medium text-muted-foreground">
          {label}
        </span>
        <div className="mt-1 flex items-center gap-2">
          {label == "创建时间" ? (
            <TimeDistance date={value as string}></TimeDistance>
          ) : (
            <span className={cn("text-sm font-medium", className)}>
              {value}
            </span>
          )}
          {/* <InfoField
            icon={icon}
            value={value}
            className={className}
          ></InfoField> */}
          {copyable && (
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 opacity-0 transition-opacity group-hover:opacity-100"
              onClick={onCopy}
            >
              <Copy className="h-3 w-3" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

// function InfoField({ icon, value, className }) {
//   if (icon == "创建时间") {
//     return <TimeDistance date={value}></TimeDistance>;
//   }
//   return <span className={cn("text-sm font-medium", className)}>{value}</span>;
// }

function KanikoInfo({ kanikoInfo: kanikoInfo, ...props }: KanikoCard) {
  const [copying, setCopying] = useState(false);
  const copyToClipboard = async (text: string) => {
    setCopying(true);
    try {
      await navigator.clipboard.writeText(text);
      toast.success("已复制到剪贴板");
    } catch (err) {
      toast.error("复制失败" + err);
    } finally {
      setCopying(false);
    }
  };
  const status = imagepackStatuses.find(
    (status) => status.value === kanikoInfo?.status,
  );
  const [namespacedName, setNamespacedName] = useState<NamespacedName>();
  return (
    <div className="w-[calc(80vw_-240px)] space-y-6">
      <Button
        type="submit"
        variant="default"
        onClick={() =>
          setNamespacedName({
            namespace: kanikoInfo?.podNameSpace ?? "",
            name: kanikoInfo?.podName ?? "",
          })
        }
      >
        日志
      </Button>
      <Card className="h-auto border-none shadow-lg">
        <CardHeader className="space-y-4 pb-4">
          <CardTitle className="flex items-center gap-2 text-xl font-semibold">
            <Info className="size-5 text-primary" />
            镜像基础信息
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg bg-card p-6 shadow-inner">
            <div className="grid gap-4 md:grid-cols-2">
              <InfoItem
                icon={<Hash className="size-4" />}
                label="ID"
                value={kanikoInfo?.ID}
              />
              <InfoItem
                icon={<FileText className="size-4" />}
                label="描述"
                value={kanikoInfo?.description}
              />
              <InfoItem
                icon={<Clock className="size-4" />}
                label="创建时间"
                value={kanikoInfo?.createdAt}
              />
              <div className="flex items-start gap-3">
                <CheckCircle className="mt-0.5 size-4 text-emerald-500" />
                <div>
                  <span className="text-sm font-medium text-muted-foreground">
                    状态:
                  </span>
                  <div className="mt-1 flex items-center gap-2">
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
                </div>
              </div>

              <InfoItem
                icon={<Link2 className="size-4" />}
                label="URL"
                value={kanikoInfo?.imageLink}
                className="break-all"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="h-auto overflow-hidden border-none shadow-lg" {...props}>
        <CardHeader className="space-y-1 pb-4">
          <CardTitle className="flex items-center gap-2 text-xl font-semibold">
            <Terminal className="size-5 text-primary" />
            Dockerfile内容
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ScrollArea className="rounded-lg">
            <div className="relative rounded-lg bg-zinc-950 p-4">
              <Button
                variant="ghost"
                size="icon"
                className="absolute right-2 top-2 h-8 w-8 text-zinc-400 hover:text-zinc-100"
                onClick={() => copyToClipboard(kanikoInfo?.dockerfile ?? "")}
                disabled={copying}
              >
                <Copy className="size-4" />
              </Button>
              <pre className="text-sm text-zinc-100">
                <code>{kanikoInfo?.dockerfile}</code>
              </pre>
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      <LogDialog
        namespacedName={namespacedName}
        setNamespacedName={setNamespacedName}
      />
    </div>
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
    setBreadcrumb([{ title: kanikoID ?? "" }]);
  }, [setBreadcrumb, kanikoID]);

  return (
    <div className="col-span-3 grid gap-8 md:grid-cols-4">
      <div className="flex-none">
        <KanikoInfo kanikoInfo={kanikoInfo} />
      </div>
    </div>
  );
};

export default KanikoDetail;
