import { Button } from "@/components/ui/button";
import { Card, CardContent, CardTitle } from "@/components/ui/card";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import {
  ActivityIcon,
  CalendarIcon,
  CheckCircleIcon,
  ClockIcon,
  CreditCardIcon,
  ExternalLinkIcon,
  FileSlidersIcon,
  FlaskConicalIcon,
  GaugeIcon,
  RefreshCcwIcon,
  TrashIcon,
  UserRoundIcon,
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
import { TimeDistance } from "@/components/custom/TimeDistance";
import { PodTable } from "./PodTable";
import { ConfigDialog } from "@/components/codeblock/ConfigDialog";
export interface Resource {
  [key: string]: string;
}

const job_monitor = import.meta.env.VITE_GRAFANA_JOB_MONITOR;
// const pod_memory = import.meta.env.VITE_GRAFANA_POD_MEMORY;
// const k8s_vgpu_scheduler_dashboard = import.meta.env
// .VITE_GRAFANA_K8S_VGPU_SCHEDULER_DASHBOARD;

export const Component = () => {
  const { name } = useParams<string>();
  const jobName = "" + name;
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
    if (!data) {
      return;
    }
    setBreadcrumb([{ title: data.name }]);
  }, [setBreadcrumb, data]);

  if (isLoading || !data) {
    return <></>;
  }

  return (
    <>
      <Card>
        <CardContent className="flex items-center justify-between p-6">
          <div className="flex items-center gap-2">
            <FlaskConicalIcon className="h-6 w-6 text-muted-foreground" />
            <CardTitle>
              {data.name}
              <span className="ml-2 text-muted-foreground">
                ({data.jobName})
              </span>
            </CardTitle>
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
              <RefreshCcwIcon className="size-4" />
            </Button>
          </div>
        </CardContent>
        <Separator />
        <CardContent className="p-6 text-sm">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
            <div className="flex items-center">
              <CreditCardIcon className="mr-2 size-4 text-muted-foreground" />
              <span className="text-sm font-medium text-muted-foreground">
                账户：
              </span>
              <span>{data.queue}</span>
            </div>
            <div className="flex items-center">
              <UserRoundIcon className="mr-2 size-4 text-muted-foreground" />
              <span className="text-sm font-medium text-muted-foreground">
                用户：
              </span>
              <span>{data.username}</span>
            </div>
            <div className="flex items-center">
              <ActivityIcon className="mr-2 size-4 text-muted-foreground" />
              <span className="text-sm font-medium text-muted-foreground">
                状态：
              </span>
              <JobPhaseLabel jobPhase={data.status} />
            </div>
            <div className="flex items-center">
              <CalendarIcon className="mr-2 size-4 text-muted-foreground" />
              <span className="text-sm font-medium text-muted-foreground">
                创建于：
              </span>
              <TimeDistance date={data.createdAt} />
            </div>
            <div className="flex items-center">
              <ClockIcon className="mr-2 size-4 text-muted-foreground" />
              <span className="text-sm font-medium text-muted-foreground">
                开始于：
              </span>
              <TimeDistance date={data.startedAt} />
            </div>
            <div className="flex items-center">
              <CheckCircleIcon className="mr-2 size-4 text-muted-foreground" />
              <span className="text-sm font-medium text-muted-foreground">
                完成于：
              </span>
              <TimeDistance date={data.completedAt} />
            </div>
          </div>
          <div className="mt-6 flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <ConfigDialog getConfig={apiJobGetYaml} jobName={jobName}>
                <Button variant="ghost" size="sm">
                  <FileSlidersIcon className="mr-1 size-4" />
                  作业 YAML
                </Button>
              </ConfigDialog>
              <Separator orientation="vertical" />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  const url = `${job_monitor}?orgId=1&var-job=${data.jobName}&from=now-1h&to=now`;
                  window.open(url, "_blank");
                }}
              >
                <GaugeIcon className="mr-1 size-4" />
                资源监控
              </Button>
            </div>
            <div className="flex items-center space-x-2">
              {(data.jobType === JobType.Jupyter ||
                data.jobType === JobType.WebIDE) &&
                data.status === JobPhase.Running && (
                  <Button
                    size="sm"
                    title="打开交互式页面"
                    onClick={() => {
                      toast.info("即将打开交互式页面");
                      setTimeout(() => {
                        getPortToken(jobName);
                      }, 500);
                    }}
                  >
                    <ExternalLinkIcon className="mr-1 size-4" />
                    交互式页面
                  </Button>
                )}
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button variant="destructive" size="sm" title="删除作业">
                      <TrashIcon className="mr-1 size-4" />
                      删除作业
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除作业</AlertDialogTitle>
                    <AlertDialogDescription>
                      作业「{data.name}」将停止，请确认已经保存好所需数据。
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
