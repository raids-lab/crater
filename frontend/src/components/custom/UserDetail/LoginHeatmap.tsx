import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import useNivoTheme from "@/hooks/useNivoTheme";
import { ResponsiveTimeRange } from "@nivo/calendar";
import { subDays, format } from "date-fns";

const generateRandomData = (days: number) => {
  const data = [];
  const today = new Date();

  for (let i = 0; i < days; i++) {
    const date = subDays(today, i);
    const formattedDate = format(date, "yyyy-MM-dd");
    const value = Math.floor(Math.random() * 10) - 7; // 生成0到5之间的随机值
    if (value < 0) {
      continue;
    }
    data.push({ day: formattedDate, value });
  }

  return data;
};
export default function LoginHeatmap() {
  const data = generateRandomData(365);
  const { nivoTheme, theme } = useNivoTheme();
  return (
    <Card>
      <CardHeader>
        <CardTitle>用户活跃度</CardTitle>
      </CardHeader>
      <CardContent>
        <div style={{ height: "200px" }}>
          <ResponsiveTimeRange
            data={data}
            from="2023-12-13"
            to="2024-12-13"
            emptyColor={theme === "dark" ? "#1f283b" : "#eeeeee"}
            colors={["#61cdbb", "#97e3d5", "#e8c1a0", "#f47560"]}
            margin={{ top: 40, right: 20, bottom: 20, left: 20 }}
            dayBorderWidth={2}
            dayBorderColor={theme === "dark" ? "#10172a" : "#ffffff"}
            firstWeekday="monday"
            theme={nivoTheme}
          />
        </div>
      </CardContent>
    </Card>
  );
}
