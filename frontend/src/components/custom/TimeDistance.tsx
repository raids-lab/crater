import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Badge } from "../ui/badge";
import { format, formatDistanceToNow } from "date-fns";
import { zhCN } from "date-fns/locale";

export const TimeDistance = ({ date }: { date: string }) => {
  if (!date) {
    return null;
  }
  const startTime = new Date(date);
  const timeDifference = formatDistanceToNow(startTime, {
    locale: zhCN,
    addSuffix: true,
  });
  return (
    <TooltipProvider delayDuration={10}>
      <Tooltip>
        <TooltipTrigger>
          <Badge
            className="border-none p-0 text-sm font-normal"
            variant="outline"
          >
            {timeDifference}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className="border bg-background text-foreground">
          {format(startTime, "PPPp", { locale: zhCN })}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
