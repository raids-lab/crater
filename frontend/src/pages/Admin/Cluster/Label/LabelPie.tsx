import { useNivoTheme } from "@/hooks/useNivoTheme";
import { apiNodeLabelsList } from "@/services/api/nodelabel";
import { ResponsivePie } from "@nivo/pie";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

export const MyResponsivePieCanvas = () => {
  const { nivoTheme, theme } = useNivoTheme();
  const query = useQuery({
    queryKey: ["label", "list"],
    queryFn: apiNodeLabelsList,
    select: (res) => res.data.data,
  });

  const data = useMemo(() => {
    if (!query.data) {
      return [];
    }
    return query.data
      .sort((a, b) => b.label.localeCompare(a.label))
      .map((item) => ({
        id: item.label !== "" ? item.label : "None",
        label: item.label !== "" ? item.label : "None",
        value: item.count,
      }));
  }, [query.data]);

  return (
    <ResponsivePie
      data={data}
      startAngle={90}
      endAngle={-270}
      margin={{ top: 25, bottom: 25 }}
      innerRadius={0.5}
      padAngle={0.7}
      cornerRadius={3}
      activeOuterRadiusOffset={8}
      colors={{ scheme: theme === "dark" ? "category10" : "set3" }}
      borderColor={{
        from: "color",
        modifiers: [["darker", 0.6]],
      }}
      theme={nivoTheme}
    />
  );
};
