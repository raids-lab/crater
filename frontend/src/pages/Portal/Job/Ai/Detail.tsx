import {
  apiAiTaskGet,
  apiJobGetLog,
  convertAiTask,
} from "@/services/api/aiTask";
import { globalBreadCrumb } from "@/utils/store";
import { useQuery } from "@tanstack/react-query";
import { useEffect, type FC } from "react";
import { useParams } from "react-router-dom";
import { useSetRecoilState } from "recoil";
import { ProgressBar } from "@/components/custom/ProgressBar";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SmallDataCard } from "@/components/custom/DataCard";

// route format: /portal/job/ai/detail?id=xxx
const AiJobDetail: FC = () => {
  const setBreadcrumb = useSetRecoilState(globalBreadCrumb);
  const { id: taskID } = useParams();

  const { data: taskInfo, isLoading } = useQuery({
    queryKey: ["aitask", "task", taskID],
    queryFn: () => apiAiTaskGet(parseInt(taskID ?? "")),
    select: (res) => convertAiTask(res.data.data),
    enabled: !!taskID,
  });

  const { data: taskLogs, isLoading: isLoadingLogs } = useQuery({
    queryKey: ["aitask", "tasklog", taskID],
    queryFn: () => apiJobGetLog(taskID ?? ""),
    select: (res) => res.data.data,
    enabled: !!taskID,
  });

  useEffect(() => {
    if (isLoading) {
      return;
    }
    setBreadcrumb([
      {
        title: "任务管理",
      },
      {
        title: "AI 训练任务",
        path: "/portal/job/ai",
      },
      {
        title: `任务详情`,
      },
    ]);
  }, [setBreadcrumb, taskInfo, isLoading]);

  if (isLoading || isLoadingLogs) {
    return <></>;
  }

  return (
    <div className="grid grid-flow-row-dense grid-cols-2 gap-6 md:grid-cols-5">
      {taskInfo && (
        <Card className="col-span-full">
          <CardHeader>
            <CardTitle>任务详情</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            {/* 包括字段有：taskName, id,nameSpace,status,slo,createdAt,startedAt,finishAt，duration，jct，image，jobName */}
            {taskInfo.taskName && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  任务名称：
                </span>
                <span className="font-mono">{taskInfo.taskName}</span>
              </div>
            )}
            {taskInfo.id && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  任务 ID：
                </span>
                <span className="font-mono">{taskInfo.id}</span>
              </div>
            )}
            {taskInfo.status && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  任务状态：
                </span>
                <span className="font-mono">{taskInfo.status}</span>
              </div>
            )}
            {taskInfo.slo && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  任务 SLO：
                </span>
                <span className="font-mono">{taskInfo.slo}</span>
              </div>
            )}
            {taskInfo.createdAt && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  创建时间：
                </span>
                {/* Date Format in YYYY-MM-DD HH:mm:ss */}
                <span className="font-mono">{taskInfo.createdAt}</span>
              </div>
            )}
            {taskInfo.startedAt && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  开始时间：
                </span>
                <span className="font-mono">{taskInfo.startedAt}</span>
              </div>
            )}
            {taskInfo.finishAt && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  结束时间：
                </span>
                <span className="font-mono">{taskInfo.finishAt}</span>
              </div>
            )}
            {taskInfo.duration !== null && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  持续时间：
                </span>
                <span className="font-mono">{taskInfo.duration}</span>
              </div>
            )}
            {taskInfo.jct && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  任务 JCT：
                </span>
                <span className="font-mono">{taskInfo.jct}</span>
              </div>
            )}
            {taskInfo.image && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  任务镜像：
                </span>
                <span className="font-mono">{taskInfo.image}</span>
              </div>
            )}
            {taskInfo.jobName && (
              <div className="col-span-full">
                <span className="inline-block w-24 text-muted-foreground">
                  Job 名称：
                </span>
                <span className="font-mono">{taskInfo.jobName}</span>
              </div>
            )}
          </CardContent>
        </Card>
      )}
      {taskInfo && taskInfo.profileStat && (
        <div className="col-span-full grid grid-cols-2 gap-4 md:grid-cols-5">
          {/* //   pcie_tx_avg: 92.72305, // 展示数值 MB/s
              //   pcie_rx_avg: 717.1082, // 展示数值 MB/s
              //   cpu_usage_avg: 1.3134052, //展示数值
              //   gpu_mem_max: 10007, //展示单位MB，上限是32768
              //   cpu_mem_max: 7325.039, // 展示单位MB，不用进度条 */}
          <SmallDataCard
            title="CPU usage avg"
            value={taskInfo.profileStat.cpu_usage_avg.toFixed(2)}
            unit="Core"
          />
          <SmallDataCard
            title="CPU mem max"
            value={taskInfo.profileStat.cpu_mem_max.toFixed(2)}
            unit="MB"
          />
          <SmallDataCard
            title="GPU mem max"
            value={`${taskInfo.profileStat.gpu_mem_max}`}
            unit="MB"
          />{" "}
          <SmallDataCard
            title="PCIE tx avg"
            value={taskInfo.profileStat.pcie_tx_avg.toFixed(2)}
            unit="MB/s"
          />
          <SmallDataCard
            title="PCIE rx avg"
            value={taskInfo.profileStat.pcie_rx_avg.toFixed(2)}
            unit="MB/s"
          />
          {/* //   gpu_util_avg: 0.97, //上限1
              //   gpu_util_max: 0.98, //上限1
              //   sm_util_avg: 0.7899093, //上限1
              //   sm_util_max: 0.798531, //上限1
              //   sm_occupancy_avg: 0.24534787, //上限1
              //   dram_util_avg: 0.24091028, //上限1
              //   mem_copy_util_avg: 0.3405, //上限1 */}
          <Card className="col-span-full">
            <CardHeader className="p-3"></CardHeader>
            <CardContent className="grid gap-3 capitalize md:grid-cols-2">
              <ProgressBar
                label={`GPU util avg: ${(
                  taskInfo.profileStat.gpu_util_avg * 100
                ).toFixed(2)}%`}
                width={taskInfo.profileStat.gpu_util_avg * 100}
              />
              <ProgressBar
                label={`GPU util max: ${(
                  taskInfo.profileStat.gpu_util_max * 100
                ).toFixed(2)}%`}
                width={taskInfo.profileStat.gpu_util_max * 100}
              />
              <ProgressBar
                label={`SM active avg: ${(
                  taskInfo.profileStat.sm_active_avg * 100
                ).toFixed(2)}%`}
                width={taskInfo.profileStat.sm_active_avg * 100}
              />
              <ProgressBar
                label={`SM active max: ${(
                  taskInfo.profileStat.sm_active_max * 100
                ).toFixed(2)}%`}
                width={taskInfo.profileStat.sm_active_max * 100}
              />
              <ProgressBar
                label={`SM occupancy avg: ${(
                  taskInfo.profileStat.sm_occupancy_avg * 100
                ).toFixed(2)}%`}
                width={taskInfo.profileStat.sm_occupancy_avg * 100}
              />
              <ProgressBar
                label={`DRAM util avg: ${(
                  taskInfo.profileStat.dram_util_avg * 100
                ).toFixed(2)}%`}
                width={taskInfo.profileStat.dram_util_avg * 100}
              />
              <ProgressBar
                label={`Mem copy util avg: ${(
                  taskInfo.profileStat.mem_copy_util_avg * 100
                ).toFixed(2)}%`}
                width={taskInfo.profileStat.mem_copy_util_avg * 100}
              />
            </CardContent>
          </Card>
        </div>
      )}
      {taskLogs && (
        <Card className="col-span-full">
          <CardHeader>
            <CardTitle>任务日志</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="whitespace-pre-wrap text-sm text-muted-foreground">
              {taskLogs}
            </pre>
          </CardContent>
        </Card>
      )}
    </div>
  );
};

export default AiJobDetail;
