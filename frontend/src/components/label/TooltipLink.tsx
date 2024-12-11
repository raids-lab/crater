import { ReactNode } from "react";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Link } from "react-router-dom";

interface TooltipLinkProps {
  name: ReactNode;
  to: string;
  tooltip: ReactNode;
  className?: string;
}

export default function TooltipLink({
  name,
  to,
  tooltip,
  className,
}: TooltipLinkProps) {
  return (
    <TooltipProvider delayDuration={50}>
      <Tooltip>
        {to.startsWith("http") ? (
          <TooltipTrigger asChild>
            <a
              href={to}
              className={cn(
                "font-normal no-underline hover:text-primary",
                className,
              )}
              target="_blank"
              rel="noopener noreferrer"
            >
              {name}
            </a>
          </TooltipTrigger>
        ) : (
          <TooltipTrigger>
            <Link
              className={cn("font-normal hover:text-primary", className)}
              to={to}
            >
              {name}
            </Link>
          </TooltipTrigger>
        )}
        <TooltipContent className="border bg-muted text-foreground">
          {tooltip}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
