import { cn } from "@/lib/utils"
import React from "react"

const CardTitle = (
  {
    ref,
    className,
    ...props
  }: React.HTMLAttributes<HTMLHeadingElement> & {
    ref?: React.RefObject<HTMLParagraphElement>;
  }
) => (<h3
  ref={ref}
  className={cn("relative font-semibold leading-none tracking-tight before:w-1 before:h-4 before:bg-primary before:absolute before:left-0 before:rounded-full pl-2.5", className)}
  {...props}
/>)
CardTitle.displayName = "CardTitle"

export { CardTitle }