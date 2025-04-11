import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useMemo } from "react"

interface MetricCardProps {
  title: string
  value: number | string
  unit: string
  description?: string
}

export function MetricCard({ title, value, unit, description }: MetricCardProps) {
  // if unit starts with "MB" and value > 1024, convert to GB
  const valueStr = useMemo(() => {
    if (typeof value === "number") {
      if (unit.startsWith("MB") && value > 1024) {
        value = value / 1024
        unit = unit.replace("MB", "GB")
      }
      return value.toFixed(2)
    } else if (typeof value === "string") {
      return value
    }
    return "0"
  }, [value, unit])

  return (
    <Card className="border">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {description && <p className="text-xs text-muted-foreground">{description}</p>}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">
          {valueStr}<span className="text-xl ml-0.5">{unit}</span>
        </div>
      </CardContent>
    </Card>
  )
}

