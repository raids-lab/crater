import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"

interface ProgressCardProps {
  title: string
  value: number
  max?: number
  unit?: string
  showPercentage?: boolean
  description?: string
}

export function ProgressCard({
  title,
  value,
  max = 1,
  unit = "",
  showPercentage = true,
  description,
}: ProgressCardProps) {
  const percentage = (value / max) * 100

  return (
    <Card className="border border-border/50">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {description && <p className="text-xs text-muted-foreground">{description}</p>}
      </CardHeader>
      <CardContent className="space-y-2">
        <div className="flex justify-between">
          <span className="text-2xl font-bold">
            {value.toFixed(2)}
            {unit}
          </span>
          {showPercentage && <span className="text-sm text-muted-foreground">{percentage.toFixed(1)}%</span>}
        </div>
        {showPercentage && <Progress value={percentage} className="h-2" />}
      </CardContent>
    </Card>
  )
}

