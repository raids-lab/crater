import type React from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Children } from "react"

interface MetricGroupProps {
  title: string
  children: React.ReactNode
}

export function MetricGroup({ title, children }: MetricGroupProps) {
  // Count the number of non-null children
  const childCount = Children.toArray(children).filter((child) => child).length

  // If there are no children, don't render the group
  if (childCount === 0) return null

  return (
    <Card className="mb-4">
      <CardHeader className="pb-2">
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">{children}</div>
      </CardContent>
    </Card>
  )
}

