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
        <TooltipTrigger disabled>
          <Badge className={cn("cursor-help", data.color)} variant="outline">
            <div>{data.label}</div>
          </Badge>
        </TooltipTrigger>
        <TooltipContent className="max-w-44">
          <p>{data.description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
