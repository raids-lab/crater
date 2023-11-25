import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { apiAiTaskGet, convertAiTask } from "@/services/api/aiTask";
import { globalBreadCrumb } from "@/utils/store";
import { useQuery } from "@tanstack/react-query";
import { useEffect, type FC } from "react";
import { useParams } from "react-router-dom";
import { useSetRecoilState } from "recoil";
import { ProgressBar } from "@/components/ProgressBar";

const SmallCard = ({
  title,
  value,
  unit,
  description,
}: {
  title: string;
  value: string;
  unit?: string;
  description?: string;
}) => {
  return (
    <Card>
      <CardHeader className="pb-1">
        <CardTitle className="text-sm font-medium capitalize">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">
          {value}
          {unit && <span className="ml-0.5 text-lg">{unit}</span>}
        </div>
        {description && (
          <p className="pt-1 text-xs text-muted-foreground">{description}</p>
        )}
      </CardContent>
    </Card>
  );
};

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

  useEffect(() => {
    if (isLoading) {
      return;
    }
    setBreadcrumb([
      {
        title: "AI 训练任务",
        path: "/portal/job/ai",
      },
      {
        title: `任务「${taskInfo?.taskName}」详情`,
      },
    ]);
  }, [setBreadcrumb, taskInfo, isLoading]);

  if (isLoading) {
    return <></>;
  }

  return (
    <div className="grid grid-flow-row-dense grid-cols-2 gap-4 md:grid-cols-5">
      {taskInfo && taskInfo.profileStat && (
        <>
          <h2 className="col-span-full text-xl font-bold">分析结果</h2>
          {/* //   pcie_tx_avg: 92.72305, // 展示数值 MB/s
              //   pcie_rx_avg: 717.1082, // 展示数值 MB/s
              //   cpu_usage_avg: 1.3134052, //展示数值
              //   gpu_mem_max: 10007, //展示单位MB，上限是32768
              //   cpu_mem_max: 7325.039, // 展示单位MB，不用进度条 */}
          <SmallCard
            title="CPU usage avg"
            value={taskInfo.profileStat.cpu_usage_avg.toFixed(2)}
            unit="Core"
          />
          <SmallCard
            title="CPU mem max"
            value={taskInfo.profileStat.cpu_mem_max.toFixed(2)}
            unit="MB"
          />
          <SmallCard
            title="GPU mem max"
            value={`${taskInfo.profileStat.gpu_mem_max}`}
            unit="MB"
          />{" "}
          <SmallCard
            title="PCIE tx avg"
            value={taskInfo.profileStat.pcie_tx_avg.toFixed(2)}
            unit="MB/s"
          />
          <SmallCard
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
            <CardContent className="grid gap-3 md:grid-cols-2">
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
                label={`SM util avg: ${(
                  taskInfo.profileStat.sm_util_avg * 100
                ).toFixed(2)}%`}
                width={taskInfo.profileStat.sm_util_avg * 100}
              />
              <ProgressBar
                label={`SM util max: ${(
                  taskInfo.profileStat.sm_util_max * 100
                ).toFixed(2)}%`}
                width={taskInfo.profileStat.sm_util_max * 100}
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
        </>
      )}
      <h2 className="col-span-full text-xl font-bold">任务数据</h2>
      <Card className="col-span-full">
        <CardHeader className="p-3"></CardHeader>
        <CardContent>
          <pre className="whitespace-pre-wrap text-sm">
            {JSON.stringify(taskInfo, null, 2)}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
};

export default AiJobDetail;
