import { apiAiTaskStats } from "@/services/api/aiTask";
import { useTheme } from "@/utils/theme";
import { useQuery } from "@tanstack/react-query";
import { useMemo, type FC } from "react";
import Chart, { Props as ChartProps } from "react-apexcharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { REFETCH_INTERVAL } from "@/config/task";
import { JobStatus } from "../Job/Ai/statuses";

const Status: FC = () => {
  const { theme } = useTheme();
  const { data: stats, isLoading } = useQuery({
    queryKey: ["aitask", "stats"],
    queryFn: apiAiTaskStats,
    select: (res) => res.data.data.taskCount,
    refetchInterval: REFETCH_INTERVAL,
  });

  const chartConfig = useMemo<ChartProps>(() => {
    const labels = ["检查中", "等待中", "运行中", "成功", "被抢占", "失败"];
    const colors = [
      "#9d59ef",
      "#677489",
      "#4ba3e3",
      "#55b685",
      "#e87b35",
      "#dd524c",
    ];
    const series = [0, 0, 0, 0, 0, 0];
    stats?.forEach((element) => {
      switch (element.Status) {
        case JobStatus.Initial:
          series[0] = element.Count;
          break;
        case JobStatus.Created:
          series[1] = element.Count;
          break;
        case JobStatus.Running:
          series[2] = element.Count;
          break;
        case JobStatus.Succeeded:
          series[3] = element.Count;
          break;
        case JobStatus.Preempted:
          series[4] = element.Count;
          break;
        case JobStatus.Failed:
          series[5] = element.Count;
          break;
        default:
          break;
      }
    });
    return {
      type: "donut",
      width: 400,
      height: 200,
      series: series,
      options: {
        chart: {
          toolbar: {
            show: false,
          },
        },
        title: {
          text: "",
        },
        dataLabels: {
          enabled: false,
        },
        labels: labels,
        colors: colors,
        legend: {
          show: true,
          position: "right",
          fontSize: "13px",
          labels: {
            colors: theme === "light" ? "#666666" : "#97a3b6",
          },
        },
        plotOptions: {
          pie: {
            donut: {},
          },
        },
        stroke: {
          colors: theme === "light" ? ["#fff"] : ["#1a1d23"],
          // width: 0,
        },
      },
    };
  }, [stats, theme]);

  return (
    <Card className="col-span-1 md:col-span-2">
      <CardHeader>
        <CardTitle>任务状态</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading || !stats || stats?.length === 0 ? (
          <p className="text-muted-foreground">暂无任务数据</p>
        ) : (
          <Chart {...chartConfig} />
        )}
      </CardContent>
    </Card>
  );
};

export default Status;
