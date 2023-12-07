import { apiAiTaskStats } from "@/services/api/aiTask";
import { useTheme } from "@/utils/theme";
import { useQuery } from "@tanstack/react-query";
import { useMemo, type FC } from "react";
import Chart, { Props as ChartProps } from "react-apexcharts";

const Status: FC = () => {
  const { theme } = useTheme();
  const { data: stats, isLoading } = useQuery({
    queryKey: ["aitask", "stats"],
    queryFn: apiAiTaskStats,
    select: (res) => res.data.data.taskCount,
  });

  const chartConfig = useMemo<ChartProps>(() => {
    // const labels = stats?.map((stat) => stat.Status) ?? [];
    // const series = stats?.map((stat) => stat.Count) ?? [];
    // const colors = labels.map((label) => {
    //   // labels: ["Queueing", "Pending", "Running", "Succeeded"],
    //   // colors: ["#ff8f00", "#d81b60", "#1e88e5", "#00897b"],
    //   switch (label) {
    //     case "Queueing":
    //       return "#9d59ef";
    //     case "Created":
    //       return "#677489";
    //     case "Pending":
    //       return "#da5597";
    //     case "Running":
    //       return "#4ba3e3";
    //     case "Failed":
    //       return "#dd524c";
    //     case "Succeeded":
    //       return "#55b685";
    //     case "Preempted":
    //       return "#e87b35";
    //     default:
    //       return "#000000";
    //   }
    // });
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

  if (!stats || stats?.length === 0) {
    return <p className="text-muted-foreground">暂无作业数据</p>;
  }

  return <Chart {...chartConfig} />;
};

export default Status;
