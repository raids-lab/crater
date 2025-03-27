import type { ReactNode } from "react"

interface MetricSectionProps {
  title: string
  icon?: ReactNode
  children: ReactNode
}

export function MetricSection({ title, icon, children }: MetricSectionProps) {
  return (
    <div className="bg-background">
      <h1 className="flex items-center gap-1.5 mb-4 text-base text-muted-foreground leading-relaxed font-bold">
        {icon}
        <span>{title}</span>
      </h1>
      {children}
    </div>
  )
}

