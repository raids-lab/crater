import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

interface MetricCardProps {
  title: string
  value: number
  unit: string
  description?: string
}

export function MetricCard({ title, value, unit, description }: MetricCardProps) {
// if unit starts with "MB" and value > 1024, convert to GB
  if (unit.startsWith("MB") && value > 1024) {
    value = value / 1024
    unit = unit.replace("MB", "GB")
  }
  return (
    <Card className="border">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {description && <p className="text-xs text-muted-foreground">{description}</p>}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">
          {value.toFixed(2)} {unit}
        </div>
      </CardContent>
    </Card>
  )
}

