import { useTheme } from "@/utils/theme";
import { useMemo, type FC } from "react";
import Chart, { Props as ChartProps } from "react-apexcharts";

const Tasks: FC = () => {
  const { theme } = useTheme();

  const chartConfig = useMemo<ChartProps>(() => {
    return {
      type: "area",
      height: 240,
      series: [
        {
          name: "Usage",
          data: [200, 100, 70, 50, 40, 300, 320, 500, 350, 200, 230, 500],
        },
      ],
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
        colors: ["#005bac"],
        stroke: {
          lineCap: "round",
          curve: "smooth",
        },
        markers: {
          size: 0,
        },
        xaxis: {
          axisTicks: {
            show: false,
          },
          axisBorder: {
            show: false,
          },
          labels: {
            style: {
              colors: "#616161",
              fontSize: "12px",
              fontFamily: "inherit",
              fontWeight: 400,
            },
          },
          categories: [
            "Jan",
            "Feb",
            "Mar",
            "Apr",
            "May",
            "Jun",
            "Jul",
            "Aug",
            "Sep",
            "Oct",
            "Nov",
            "Dec",
          ],
        },
        yaxis: {
          labels: {
            style: {
              colors: "#616161",
              fontSize: "12px",
              fontFamily: "inherit",
              fontWeight: 400,
            },
          },
        },
        grid: {
          show: true,
          borderColor: theme === "light" ? "#dddddd" : "#20293a",
          strokeDashArray: 5,
          xaxis: {
            lines: {
              show: true,
            },
          },
          padding: {
            top: 5,
            right: 20,
          },
        },
        fill: {
          opacity: 0.8,
        },
        tooltip: {
          theme: theme,
          // x: {
          //   show: false,
          // },
          // y: {
          //   show: false,
          // },
        },
      },
    };
  }, [theme]);

  return <Chart {...chartConfig} />;
};

export default Tasks;
