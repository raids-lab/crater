import { Button } from "@/components/ui/button";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import {
  ActivityIcon,
  BarChartBigIcon,
  CalendarIcon,
  CheckCircleIcon,
  ClockIcon,
  CreditCardIcon,
  ExternalLinkIcon,
  FileSlidersIcon,
  GaugeIcon,
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
import { useEffect, useMemo } from "react";
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
  apiSSHPortGetDetail,
} from "@/services/api/vcjob";
import JobPhaseLabel from "@/components/badge/JobPhaseBadge";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { PodTable } from "./PodTable";
import { CodeContent } from "@/components/codeblock/ConfigDialog";
import { LazyContent } from "@/components/codeblock/Dialog";
import { EventTimeline } from "@/components/custom/Timeline/timeline-layout";
import PageTitle from "@/components/layout/PageTitle";
import JobTypeLabel from "@/components/badge/JobTypeBadge";
import { GrafanaIframe } from "@/pages/Embed/Monitor";
import useFixedLayout from "@/hooks/useFixedLayout";
import { DetailPage } from "@/components/layout/DetailPage";
import Nothing from "@/components/placeholder/Nothing";
import { hasNvidiaGPU } from "@/utils/resource";
import GpuIcon from "@/components/icon/GpuIcon";
import { SSHPortDialog } from "./SSHPortDialog";
import ProfileDashboard from "@/components/profile-dashboard";
import { getDaysDifference, getHoursDifference } from "@/utils/time";
export interface Resource {
  [key: string]: string;
}

const job_monitor = import.meta.env.VITE_GRAFANA_JOB_MONITOR;
const job_gpu_monitor = import.meta.env.VITE_GRAFANA_JOB_GPU_MONITOR;

export function BaseCore({ jobName }: { jobName: string }) {
  useFixedLayout();
  const navigate = useNavigate();

  const { data, isLoading } = useQuery({
    queryKey: ["job", "detail", jobName],
    queryFn: () => apiJobGetDetail(jobName),
    select: (res) => res.data.data,
  });

  const { data: sshPortData } = useQuery({
    queryKey: ["job", "ssh", jobName],
    queryFn: () => apiSSHPortGetDetail(jobName),
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

  const showGPUDashboard = useMemo(() => {
    if (!data) {
      return false;
    }
    return hasNvidiaGPU(data.resources);
  }, [data]);

  const profileStat = useMemo(() => {
    if (!data) {
      return null;
    }
    return data.profileData;
  }, [data]);

  const isCompletedOver3Days = useMemo(() => {
    return getDaysDifference(data?.completedAt) > 3;
  }, [data?.completedAt]);

  const isCompletedOver1Hour = useMemo(() => {
    return getHoursDifference(data?.completedAt) > 1 / 24;
  }, [data?.completedAt]);

  if (isLoading || !data) {
    return <></>;
  }

  return (
    <DetailPage
      header={
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
              data.status === JobPhase.Running &&
              sshPortData &&
              sshPortData.open && (
                <SSHPortDialog
                  hostIP={sshPortData.data.IP}
                  nodePort={sshPortData.data.nodePort}
                  userName={sshPortData.data.username}
                />
              )}
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
      }
      info={[
        { title: "账户", icon: CreditCardIcon, value: data.queue },
        { title: "用户", icon: UserRoundIcon, value: data.username },
        {
          title: "状态",
          icon: ActivityIcon,
          value: <JobPhaseLabel jobPhase={data.status} />,
        },
        {
          title: "创建于",
          icon: CalendarIcon,
          value: <TimeDistance date={data.createdAt} />,
        },
        {
          title: "开始于",
          icon: ClockIcon,
          value: <TimeDistance date={data.startedAt} />,
        },
        {
          title: "完成于",
          icon: CheckCircleIcon,
          value: <TimeDistance date={data.completedAt} />,
        },
      ]}
      tabs={[
        {
          key: "profile",
          icon: BarChartBigIcon,
          label: "统计信息",
          children: <ProfileDashboard profileData={data.profileData ?? {}} />,
          scrollable: true,
          hidden: !profileStat,
        },
        {
          key: "base",
          icon: LayoutGridIcon,
          label: "基本信息",
          children: (
            <>
              <PodTable jobName={jobName} />
            </>
          ),
          scrollable: true,
          hidden:
            data.status === JobPhase.Deleted ||
            data.status === JobPhase.Freed ||
            isCompletedOver3Days,
        },
        {
          key: "config",
          icon: FileSlidersIcon,
          label: "作业配置",
          children: (
            <LazyContent
              name={jobName}
              type="yaml"
              fetchData={apiJobGetYaml}
              renderData={(data) => (
                <CodeContent data={data} language={"yaml"} />
              )}
            />
          ),
        },
        {
          key: "event",
          icon: ShieldEllipsisIcon,
          label: "作业事件",
          children: (
            <LazyContent
              name={jobName}
              type="event"
              fetchData={apiJobGetEvent}
              renderData={(events) => (
                <>
                  <EventTimeline items={events} />
                  {events.length === 0 && (
                    <Nothing title={"近一小时无事发生"} />
                  )}
                </>
              )}
            />
          ),
          scrollable: true,
          hidden: isCompletedOver1Hour,
        },
        {
          key: "monitor",
          icon: GaugeIcon,
          label: "基础监控",
          children: (
            <div className="h-[calc(100vh_-_304px)] w-full">
              <GrafanaIframe
                baseSrc={`${job_monitor}?var-job=${data.jobName}&from=now-1h&to=now`}
              />
            </div>
          ),
        },
        {
          key: "gpu",
          icon: GpuIcon,
          label: "加速卡监控",
          children: (
            <div className="h-[calc(100vh_-_304px)] w-full">
              <GrafanaIframe
                baseSrc={`${job_gpu_monitor}?var-job=${data.jobName}&from=now-1h&to=now`}
              />
            </div>
          ),
          hidden: !showGPUDashboard,
        },
      ]}
    />
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
