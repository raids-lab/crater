import { apiAiTaskStats } from "@/services/api/aiTask";
import { useTheme } from "@/utils/theme";
import { useQuery } from "@tanstack/react-query";
import { useMemo, type FC } from "react";
import Chart, { Props as ChartProps } from "react-apexcharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { REFETCH_INTERVAL } from "@/config/task";

const Status: FC = () => {
  const { theme } = useTheme();
  const { data: stats, isLoading } = useQuery({
    queryKey: ["aitask", "stats"],
    queryFn: apiAiTaskStats,
    select: (res) => res.data.data.taskCount,
    refetchInterval: REFETCH_INTERVAL,
  });

  const chartConfig = useMemo<ChartProps>(() => {
    const labels = [
      "Queueing",
      "Created",
      "Pending",
      "Running",
      "Failed",
      "Succeeded",
      "Preempted",
    ];
    const colors = [
      "#9d59ef",
      "#677489",
      "#da5597",
      "#4ba3e3",
      "#dd524c",
      "#55b685",
      "#e87b35",
    ];
    const series = [0, 0, 0, 0, 0, 0, 0];
    stats?.forEach((element) => {
      switch (element.Status) {
        case "Queueing":
          series[0] = element.Count;
          break;
        case "Created":
          series[1] = element.Count;
          break;
        case "Pending":
          series[2] = element.Count;
          break;
        case "Running":
          series[3] = element.Count;
          break;
        case "Failed":
          series[4] = element.Count;
          break;
        case "Succeeded":
          series[5] = element.Count;
          break;
        case "Preempted":
          series[6] = element.Count;
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
          colors: theme === "light" ? ["#fff"] : ["#000"],
        },
      },
    };
  }, [stats, theme]);

  if (isLoading) {
    return <></>;
  }

  return (
    <Card className="col-span-1">
      <CardHeader>
        <CardTitle>任务状态</CardTitle>
      </CardHeader>
      <CardContent>
        {!stats || stats?.length === 0 ? (
          <p className="text-muted-foreground">暂无作业数据</p>
        ) : (
          <Chart {...chartConfig} />
        )}
      </CardContent>
    </Card>
  );
};

export default Status;
