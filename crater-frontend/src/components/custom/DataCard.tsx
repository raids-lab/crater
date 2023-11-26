import { useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export const SmallDataCard = ({
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
  const [newValue, newUnit] = useMemo(() => {
    // 10007MB -> 9.54GB
    if (unit === "MB" && parseFloat(value) > 1024) {
      return [(parseFloat(value) / 1024).toFixed(2), "GB"];
    }
    // 10007MB/s -> 9.54GB/s
    if (unit === "MB/s" && parseFloat(value) > 1024) {
      return [(parseFloat(value) / 1024).toFixed(2), "GB/s"];
    }
    return [value, unit];
  }, [value, unit]);
  return (
    <Card>
      <CardHeader className="pb-0">
        <CardTitle className="text-sm font-medium capitalize">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">
          {newValue}
          {newUnit && <span className="ml-0.5 text-lg">{newUnit}</span>}
        </div>
        {description && (
          <p className="pt-1 text-xs text-muted-foreground">{description}</p>
        )}
      </CardContent>
    </Card>
  );
};
