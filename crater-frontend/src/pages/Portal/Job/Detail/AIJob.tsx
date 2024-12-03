import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import {
  FileSlidersIcon,
  PieChartIcon,
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
import { useEffect, useMemo, useState } from "react";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";
import {
  JobPhase,
  apiJobDelete,
  apiJobGetDetail,
  apiJobGetYaml,
  PodDetail,
} from "@/services/api/vcjob";
import JobPhaseLabel from "@/components/label/JobPhaseLabel";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { SmallDataCard } from "@/components/custom/DataCard";
import { ProfileStat } from "@/services/api/aiTask";
import { ConfigDialog } from "@/components/codeblock/ConfigDialog";
import { PodTable } from "./PodTable";

export interface Resource {
  [key: string]: string;
}

interface AIjobDetail {
  name: string;
  namespace: string;
  username: string;
  jobName: string;
  retry: string;
  queue: string;
  status: JobPhase;
  createdAt: string;
  startedAt: string;
  runtime: string;
  podDetails: PodDetail[];
  useTensorBoard: boolean;
  id: number;
  priority: string;
  profileStat: string;
  profileStatus: string;
}

const initial: AIjobDetail = {
  name: "",
  namespace: "",
  username: "",
  jobName: "",
  retry: "",
  queue: "",
  status: JobPhase.Unknown,
  createdAt: "",
  startedAt: "",
  runtime: "",
  podDetails: [],
  useTensorBoard: false,
  id: 0,
  priority: "",
  profileStat: "",
  profileStatus: "",
};

const job_monitor = import.meta.env.VITE_GRAFANA_JOB_MONITOR;

export const Component = () => {
  const { name } = useParams<string>();
  const jobName = "" + name;
  const setBreadcrumb = useBreadcrumb();
  const [data, setData] = useState<AIjobDetail>(initial);
  const [refetchInterval, setRefetchInterval] = useState(5000); // Manage interval state
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const { data: taskInfo, isLoading } = useQuery({
    queryKey: ["job", "detail", jobName],
    queryFn: () => apiJobGetDetail(jobName),
    select: (res) => res.data.data as unknown as AIjobDetail,
    refetchInterval: refetchInterval,
  });

  const profileStat = useMemo(() => {
    if (taskInfo?.profileStat) {
      return JSON.parse(taskInfo.profileStat) as ProfileStat;
    }
    return null;
  }, [taskInfo?.profileStat]);

  const handleIntervalChange = (newInterval: string) => {
    const intervalMs = parseInt(newInterval) * 1000; // Convert seconds to milliseconds
    setRefetchInterval(intervalMs);
  };

  const { mutate: deleteJTask } = useMutation({
    mutationFn: () => apiJobDelete(jobName),
    onSuccess: () => {
      navigate("/jupyter");
      toast.success("作业已删除");
    },
  });

  useEffect(() => {
    setBreadcrumb([{ title: "作业详情" }]);
  }, [setBreadcrumb]);

  useEffect(() => {
    if (!isLoading && taskInfo) {
      setData(taskInfo);
    }
  }, [taskInfo, isLoading, taskInfo?.runtime]);

  return (
    <>
      <Card className="col-span-3">
        <CardContent className="flex items-center justify-between bg-muted/50 p-6">
          <div className="flex flex-row items-center gap-2">
            <h1 className="text-xl font-semibold capitalize text-foreground">
              {data.name}
            </h1>
          </div>
          <div className="flex items-center gap-2">
            <Select onValueChange={(value) => handleIntervalChange(value)}>
              <SelectTrigger className="h-8 w-[130px] bg-background">
                <SelectValue placeholder="每 5s 刷新" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="5">每 5s 刷新</SelectItem>
                <SelectItem value="10">每 10s 刷新</SelectItem>
                <SelectItem value="20">每 20s 刷新</SelectItem>
              </SelectContent>
            </Select>
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
                <TableHead className="text-xs">命名空间</TableHead>
                <TableHead className="text-xs">创建于</TableHead>
                <TableHead className="text-xs">持续时间</TableHead>
                <TableHead className="text-xs">重试次数</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>{data.jobName}</TableCell>
                <TableCell>
                  <JobPhaseLabel jobPhase={data.status} />
                </TableCell>
                <TableCell>{data.queue}</TableCell>
                <TableCell>{data.username}</TableCell>
                <TableCell>{data.namespace}</TableCell>
                <TableCell>
                  <TimeDistance date={data.createdAt} />
                </TableCell>
                <TableCell>{data.runtime}</TableCell>
                <TableCell>{data.retry}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
          <div className="mt-4 flex items-center justify-between">
            <div className="flex h-5 items-center space-x-2">
              <ConfigDialog getConfig={apiJobGetYaml} jobName={jobName}>
                <Button variant="ghost" size="sm">
                  <FileSlidersIcon className="h-4 w-4" />
                  作业 YAML
                </Button>
              </ConfigDialog>
              <Separator orientation="vertical" />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  // const url = generateMetricsUrl(data.podDetails);
                  const url = `${job_monitor}?orgId=1&var-job=${data.jobName}&from=now-1h&to=now`;
                  window.open(url, "_blank");
                }}
              >
                <PieChartIcon className="h-4 w-4" />
                资源监控
              </Button>
            </div>
            <div className="flex flex-row gap-2">
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button variant="destructive" size="sm" title="删除作业">
                      <TrashIcon className="h-4 w-4" />
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
      {profileStat && (
        <Card className="col-span-full">
          <CardHeader>
            <CardTitle>Profile 结果</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 capitalize md:grid-cols-3">
            <SmallDataCard
              title="CPU usage avg"
              value={profileStat.cpu_usage_avg.toFixed(2)}
              unit="Core"
            />
            <SmallDataCard
              title="CPU mem max"
              value={profileStat.cpu_mem_max.toFixed(2)}
              unit="MB"
            />
            <SmallDataCard
              title="GPU mem max"
              value={`${profileStat.gpu_mem_max}`}
              unit="MB"
            />{" "}
            <SmallDataCard
              title="PCIE tx avg"
              value={profileStat.pcie_tx_avg.toFixed(2)}
              unit="MB/s"
            />
            <SmallDataCard
              title="PCIE rx avg"
              value={profileStat.pcie_rx_avg.toFixed(2)}
              unit="MB/s"
            />
            <SmallDataCard
              title="GPU util avg"
              value={(profileStat.gpu_util_avg * 100).toFixed(2)}
            />
            <SmallDataCard
              title="GPU util max"
              value={(profileStat.gpu_util_max * 100).toFixed(2)}
            />
            <SmallDataCard
              title="SM active avg"
              value={(profileStat.sm_active_avg * 100).toFixed(2)}
            />
            <SmallDataCard
              title="SM active max"
              value={(profileStat.sm_active_max * 100).toFixed(2)}
            />
            <SmallDataCard
              title="SM occupancy avg"
              value={(profileStat.sm_occupancy_avg * 100).toFixed(2)}
            />
            <SmallDataCard
              title="DRAM util avg"
              value={(profileStat.dram_util_avg * 100).toFixed(2)}
            />
            <SmallDataCard
              title="Mem copy util avg"
              value={(profileStat.mem_copy_util_avg * 100).toFixed(2)}
            />
          </CardContent>
        </Card>
      )}
      <PodTable jobName={jobName} />
    </>
  );
};
