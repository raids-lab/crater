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
                "hover:text-primary font-normal no-underline",
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
              className={cn("hover:text-primary font-normal", className)}
              to={to}
            >
              {name}
            </Link>
          </TooltipTrigger>
        )}
        <TooltipContent>{tooltip}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
