/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ProgressBar } from "./colorful-progress"

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
        {percentage > 0.1 && <ProgressBar percent={percentage} />}
      </CardContent>
    </Card>
  )
}

