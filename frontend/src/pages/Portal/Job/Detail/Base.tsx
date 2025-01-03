import { Button } from "@/components/ui/button";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import {
  ActivityIcon,
  CalendarIcon,
  CheckCircleIcon,
  ClockIcon,
  CreditCardIcon,
  ExternalLinkIcon,
  FileSlidersIcon,
  GaugeIcon,
  GridIcon,
  LayoutGridIcon,
  ShieldEllipsisIcon,
  Trash2Icon,
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
import { useQuery, useMutation } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import {
  apiJupyterTokenGet,
  apiJobDelete,
  apiJobGetDetail,
  apiJobGetYaml,
  JobType,
  JobPhase,
  apiJobGetEvent,
} from "@/services/api/vcjob";
import JobPhaseLabel from "@/components/badge/JobPhaseBadge";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { PodTable } from "./PodTable";
import { CodeContent } from "@/components/codeblock/ConfigDialog";
import { LazyContent } from "@/components/codeblock/Dialog";
import { EventTimeline } from "@/components/custom/Timeline/timeline-layout";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import PageTitle from "@/components/layout/PageTitle";
import JobTypeLabel from "@/components/badge/JobTypeBadge";
import { GrafanaIframe } from "@/pages/Embed/Monitor";
export interface Resource {
  [key: string]: string;
}

const job_monitor = import.meta.env.VITE_GRAFANA_JOB_MONITOR;
// const pod_memory = import.meta.env.VITE_GRAFANA_POD_MONITOR;
// const k8s_vgpu_scheduler_dashboard = import.meta.env
// .VITE_GRAFANA_GPU_DASHBOARD;

export function BaseCore({ jobName }: { jobName: string }) {
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

  if (isLoading || !data) {
    return <></>;
  }

  return (
    <div className="h-[calc(100vh_-_80px)] w-full space-y-6">
      <div className="space-y-6">
        <PageTitle
          title={
            <div className="flex flex-row items-center gap-1.5 text-2xl">
              {data.name}
              <JobTypeLabel jobType={data.jobType} />
            </div>
          }
          description={data.jobName}
        >
          <div className="flex flex-row gap-3">
            {(data.jobType === JobType.Jupyter ||
              data.jobType === JobType.WebIDE) &&
              data.status === JobPhase.Running && (
                <Button
                  variant="secondary"
                  title="打开交互式页面"
                  onClick={() => {
                    toast.info("即将打开交互式页面");
                    setTimeout(() => {
                      getPortToken(jobName);
                    }, 500);
                  }}
                >
                  <ExternalLinkIcon className="size-4" />
                  交互式页面
                </Button>
              )}
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <div>
                  <Button variant="destructive" title="删除作业">
                    <Trash2Icon className="size-4" />
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
        </PageTitle>
        <div className="grid grid-cols-2 gap-4 text-sm text-muted-foreground lg:grid-cols-3">
          <div className="flex items-center">
            <CreditCardIcon className="mr-1.5 size-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">账户：</span>
            <span>{data.queue}</span>
          </div>
          <div className="flex items-center">
            <UserRoundIcon className="mr-1.5 size-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">用户：</span>
            <span>{data.username}</span>
          </div>
          <div className="flex items-center">
            <ActivityIcon className="mr-1.5 size-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">状态：</span>
            <JobPhaseLabel jobPhase={data.status} />
          </div>
          <div className="flex items-center">
            <CalendarIcon className="mr-1.5 size-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">创建于：</span>
            <TimeDistance date={data.createdAt} />
          </div>
          <div className="flex items-center">
            <ClockIcon className="mr-1.5 size-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">开始于：</span>
            <TimeDistance date={data.startedAt} />
          </div>
          <div className="flex items-center">
            <CheckCircleIcon className="mr-1.5 size-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">完成于：</span>
            <TimeDistance date={data.completedAt} />
          </div>
        </div>
      </div>
      <Tabs defaultValue="base" className="w-full">
        <TabsList className="tabs-list-underline">
          <TabsTrigger className="tabs-trigger-underline" value="base">
            <LayoutGridIcon className="size-4" />
            基本信息
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="config">
            <FileSlidersIcon className="size-4" />
            作业配置
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="event">
            <ShieldEllipsisIcon className="size-4" />
            作业事件
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="monitor">
            <GaugeIcon className="size-4" />
            资源监控
          </TabsTrigger>
        </TabsList>
        <TabsContent value="base">
          <PodTable jobName={jobName} />
        </TabsContent>
        <TabsContent value="config">
          <LazyContent
            name={jobName}
            type="yaml"
            fetchData={apiJobGetYaml}
            renderData={(data) => <CodeContent data={data} language={"yaml"} />}
          />
        </TabsContent>
        <TabsContent value="event">
          <LazyContent
            name={jobName}
            type="event"
            fetchData={apiJobGetEvent}
            renderData={(events) => (
              <>
                <EventTimeline items={events} />
                {events.length === 0 && (
                  <div className="flex h-[calc(100vh-_304px)] w-full items-center justify-center text-center text-muted-foreground/85 hover:bg-transparent">
                    <div className="flex flex-col items-center justify-center">
                      <div className="mb-4 rounded-full bg-muted p-3">
                        <GridIcon className="h-6 w-6" />
                      </div>
                      <p className="select-none text-sm">近一小时无事发生</p>
                    </div>
                  </div>
                )}
              </>
            )}
          />
        </TabsContent>
        <TabsContent value="monitor">
          <div className="h-[calc(100vh_-_304px)] w-full">
            <GrafanaIframe
              baseSrc={`${job_monitor}?orgId=1&var-job=${data.jobName}&from=now-1h&to=now`}
            />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}

export const Component = () => {
  const { name } = useParams<string>();
  const jobName = "" + name;
  const setBreadcrumb = useBreadcrumb();

  useEffect(() => {
    setBreadcrumb([{ title: "作业详情" }]);
  }, [setBreadcrumb]);

  return <BaseCore jobName={jobName} />;
};
