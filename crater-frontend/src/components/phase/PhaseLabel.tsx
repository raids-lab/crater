import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export interface PhaseLabelData {
  label: string;
  color: string;
  description: string;
}

interface PhaseLabelProps<T> {
  phase: T;
  getPhaseLabel: (phase: T) => PhaseLabelData;
}

export const PhaseLabel = <T,>({
  phase,
  getPhaseLabel,
}: PhaseLabelProps<T>) => {
  const data = getPhaseLabel(phase);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger disabled>
          <Badge className={cn("cursor-pointer", data.color)} variant="outline">
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
