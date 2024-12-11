import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useNivoTheme } from "@/hooks/useNivoTheme";
import { ResponsiveCalendar } from "@nivo/calendar";

const data = [
  { day: "2024-03-01", value: 3 },
  { day: "2024-03-02", value: 5 },
  { day: "2024-03-03", value: 2 },
  { day: "2024-05-21", value: 4 },
  // ... 添加更多数据点
];

export default function LoginHeatmap() {
  const { nivoTheme, theme } = useNivoTheme();
  return (
    <Card>
      <CardHeader>
        <CardTitle>用户活跃度</CardTitle>
      </CardHeader>
      <CardContent>
        <div style={{ height: "400px" }}>
          <ResponsiveCalendar
            data={data}
            from="2023-12-11"
            to="2024-12-11"
            emptyColor={theme === "dark" ? "#2d3748" : "#eeeeee"}
            colors={["#61cdbb", "#97e3d5", "#e8c1a0", "#f47560"]}
            margin={{ top: 20, right: 20, bottom: 20, left: 20 }}
            yearSpacing={40}
            monthSpacing={16}
            monthLegend={(_, month) => `${month + 1}月`}
            monthBorderColor={theme === "dark" ? "#1a202c" : "#fdfdfd"}
            dayBorderWidth={2}
            dayBorderColor={theme === "dark" ? "#0d1017" : "#ffffff"}
            theme={nivoTheme}
          />
        </div>
      </CardContent>
    </Card>
  );
}
