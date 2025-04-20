import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export interface PhaseBadgeData {
  label: string;
  color: string;
  description: string;
}

interface PhaseBadgeProps<T> {
  phase: T;
  getPhaseLabel: (phase: T) => PhaseBadgeData;
}

export const PhaseBadge = <T,>({
  phase,
  getPhaseLabel,
}: PhaseBadgeProps<T>) => {
  const data = getPhaseLabel(phase);

  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge
            className={cn("cursor-help border-none", data.color)}
            variant="outline"
          >
            <div>{data.label}</div>
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          <p>{data.description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
