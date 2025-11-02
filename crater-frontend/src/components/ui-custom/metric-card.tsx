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

