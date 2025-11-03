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

