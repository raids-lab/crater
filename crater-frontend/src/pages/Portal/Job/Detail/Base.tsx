import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import {
  ExternalLinkIcon,
  FileSlidersIcon,
  GaugeIcon,
  RefreshCcwIcon,
  TrashIcon,
} from "lucide-react";
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
import { useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import {
  TableHead,
  TableRow,
  TableHeader,
  TableCell,
  TableBody,
  Table,
} from "@/components/ui/table";
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";
import {
  apiJupyterTokenGet,
  apiJobDelete,
  apiJobGetDetail,
  apiJobGetYaml,
  JobType,
  JobPhase,
} from "@/services/api/vcjob";
import JobPhaseLabel from "@/components/label/JobPhaseLabel";
import { TableDate } from "@/components/custom/TableDate";
import { PodTable } from "./PodTable";
import { ConfigDialog } from "@/components/codeblock/ConfigDialog";
import { CardTitle } from "@/components/ui-custom/card";
export interface Resource {
  [key: string]: string;
}

const job_monitor = import.meta.env.VITE_GRAFANA_JOB_MONITOR;
// const pod_memory = import.meta.env.VITE_GRAFANA_POD_MEMORY;
// const k8s_vgpu_scheduler_dashboard = import.meta.env
// .VITE_GRAFANA_K8S_VGPU_SCHEDULER_DASHBOARD;

export const Component = () => {
  const { id } = useParams<string>();
  const jobName = "" + id;
  const setBreadcrumb = useBreadcrumb();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const { data, isLoading } = useQuery({
    queryKey: ["job", "detail", jobName],
    queryFn: () => apiJobGetDetail(jobName),
    select: (res) => res.data.data,
  });

  const { mutate: getPortToken } = useMutation({
    mutationFn: (jobName: string) => apiJupyterTokenGet(jobName),
    onSuccess: (_, jobName) => {
      window.open(`/job/jupyter/${jobName}`);
    },
  });

  const { mutate: deleteJTask } = useMutation({
    mutationFn: () => apiJobDelete(jobName),
    onSuccess: () => {
      navigate(-1);
      toast.success("作业已删除");
    },
  });

  useEffect(() => {
    setBreadcrumb([{ title: "作业详情" }]);
  }, [setBreadcrumb]);

  if (isLoading || !data) {
    return <div>Loading...</div>;
  }

  return (
    <>
      <Card className="col-span-3">
        <CardContent className="flex items-center justify-between bg-muted/50 p-6 dark:bg-muted/25">
          <div className="flex flex-col items-start gap-2">
            <CardTitle>{data.name}</CardTitle>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() =>
                void queryClient.invalidateQueries({
                  queryKey: ["job", "detail", jobName],
                })
              }
            >
              <RefreshCcwIcon className="h-3.5 w-3.5" />
            </Button>
          </div>
        </CardContent>
        <Separator />
        <CardContent className="flex flex-col gap-4 p-7">
          <Table className="w-full">
            <TableHeader>
              <TableRow>
                <TableHead className="text-xs">作业名称</TableHead>
                <TableHead className="text-xs">状态</TableHead>
                <TableHead className="text-xs">账户</TableHead>
                <TableHead className="text-xs">用户</TableHead>
                <TableHead className="text-xs">创建于</TableHead>
                <TableHead className="text-xs">开始于</TableHead>
                <TableHead className="text-xs">完成于</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell className="font-mono">{data.jobName}</TableCell>
                <TableCell>
                  <JobPhaseLabel jobPhase={data.status} />
                </TableCell>
                <TableCell>{data.queue}</TableCell>
                <TableCell>{data.username}</TableCell>
                <TableCell>
                  <TableDate date={data.createdAt} />
                </TableCell>
                <TableCell>
                  <TableDate date={data.startedAt} />
                </TableCell>
                <TableCell>
                  <TableDate date={data.completedAt} />
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
          <div className="mt-4 flex items-center justify-between">
            <div className="flex h-5 items-center space-x-2">
              <ConfigDialog getConfig={apiJobGetYaml} jobName={jobName}>
                <Button variant="ghost" size="sm">
                  <FileSlidersIcon className="mr-2 h-4 w-4" />
                  作业 YAML
                </Button>
              </ConfigDialog>
              <Separator orientation="vertical" />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  // const url = generateMetricsUrl(data.podDetails);
                  const url = `${job_monitor}?orgId=1&var-job=${data.jobName}`;
                  window.open(url, "_blank");
                }}
              >
                <GaugeIcon className="mr-2 h-4 w-4" />
                资源监控
              </Button>
            </div>
            <div className="flex flex-row gap-2">
              {(data.jobType === JobType.Jupyter ||
                data.jobType === JobType.WebIDE) &&
                data.status === JobPhase.Running && (
                  <Button
                    size="sm"
                    title="跳转至交互式页面"
                    onClick={() => {
                      toast.info("即将跳转至交互式页面");
                      setTimeout(() => {
                        getPortToken(jobName);
                      }, 500);
                    }}
                  >
                    <ExternalLinkIcon className="mr-2 h-4 w-4" />
                    交互式页面
                  </Button>
                )}
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button variant="destructive" size="sm" title="删除作业">
                      <TrashIcon className="mr-2 h-4 w-4" />
                      删除作业
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除作业</AlertDialogTitle>
                    <AlertDialogDescription>
                      作业「{data.name}
                      」将停止，请确认已经保存好所需数据。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        deleteJTask();
                      }}
                    >
                      删除
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          </div>
        </CardContent>
      </Card>
      <PodTable jobName={jobName} />
    </>
  );
};
