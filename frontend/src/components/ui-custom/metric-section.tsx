import { cn } from "@/lib/utils"
import type { ReactNode } from "react"

interface MetricSectionProps {
  title: string
  icon?: ReactNode
  children: ReactNode
  className?: string
}

export function MetricSection({ title, icon, children, className }: MetricSectionProps) {
  return (
    <div className="bg-background">
      <h1 className="flex items-center gap-1.5 mb-4 text-base text-muted-foreground leading-relaxed font-bold">
        {icon}
        <span>{title}</span>
      </h1>
      <div className={cn("grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3", className)}>
      {children}
      </div>
    </div>
  )
}

