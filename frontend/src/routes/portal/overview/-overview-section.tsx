import type { LucideIcon } from 'lucide-react'
import { type ReactNode, useId } from 'react'

interface OverviewSectionProps {
  title: string
  description: string
  icon: LucideIcon
  accent: string
  action?: ReactNode
  children: ReactNode
}

export function OverviewSection({
  title,
  description,
  icon: Icon,
  accent,
  action,
  children,
}: OverviewSectionProps) {
  const headingId = useId()

  return (
    <section
      aria-labelledby={headingId}
      className="border-border bg-card overflow-hidden rounded-xl border shadow-xs"
    >
      <header className="border-border bg-muted/25 flex flex-wrap items-center justify-between gap-3 border-b px-4 py-4 sm:px-5">
        <div className="flex min-w-0 items-center gap-3">
          <div className={`grid size-10 shrink-0 place-items-center rounded-lg ${accent}`}>
            <Icon className="size-5" aria-hidden="true" />
          </div>
          <div className="min-w-0">
            <h2 id={headingId} className="text-base font-semibold tracking-tight">
              {title}
            </h2>
            <p className="text-muted-foreground mt-0.5 text-sm">{description}</p>
          </div>
        </div>
        {action}
      </header>
      <div className="min-w-0 px-4 py-4 sm:px-5">{children}</div>
    </section>
  )
}
