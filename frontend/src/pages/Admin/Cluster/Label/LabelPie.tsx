import NivoPie from "@/components/chart/NivoPie";
import { apiNodeLabelsList } from "@/services/api/nodelabel";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

export const MyResponsivePieCanvas = () => {
  const query = useQuery({
    queryKey: ["label", "list"],
    queryFn: apiNodeLabelsList,
    select: (res) =>
      res.data.data
        .sort((a, b) => b.label.localeCompare(a.label))
        .map((item) => ({
          id: item.label !== "" ? item.label : "None",
          label: item.label !== "" ? item.label : "None",
          value: item.count,
        })),
  });

  const data = useMemo(() => {
    if (!query.data) {
      return [];
    }
    return query.data;
  }, [query.data]);

  return (
    <NivoPie
      data={data}
      startAngle={90}
      endAngle={-270}
      margin={{ top: 25, bottom: 25 }}
    />
  );
};
