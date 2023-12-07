import type { FC } from "react";
import Chart, { Props as ChartProps } from "react-apexcharts";

const chartConfig: ChartProps = {
  type: "donut",
  width: 320,
  height: 200,
  series: [44, 13, 43, 22],
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
    labels: ["Queueing", "Pending", "Running", "Succeeded"],
    colors: ["#ff8f00", "#d81b60", "#1e88e5", "#00897b"],
    legend: {
      show: true,
      position: "left",
    },
  },
};

const Status: FC = () => {
  return <Chart {...chartConfig} />;
};

export default Status;
