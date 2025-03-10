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
      <CardContent className="space-y-2">
        <div className="flex justify-between">
          {!showPercentage && 
            (<span className="text-2xl font-bold">
              {value.toFixed(2)}
              <span className="text-xl ml-0.5">{unit}</span>
            </span>)}
          {showPercentage && <span className="text-2xl font-bold">{percentage.toFixed(1)}<span className="text-xl ml-0.5">%</span></span>}
        </div>
        {percentage > 0 && <Progress value={percentage} className="h-2" />}
      </CardContent>
    </Card>
  )
}

