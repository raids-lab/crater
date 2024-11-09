import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Badge } from "../ui/badge";
import { format, formatDistanceToNow } from "date-fns";
import { zhCN } from "date-fns/locale";
import { useMemo } from "react";

export const TimeDistance = ({ date }: { date: string }) => {
  const [startTime, timeDiff] = useMemo(() => {
    const startTime = new Date(date);
    const timeDifference = formatDistanceToNow(startTime, {
      locale: zhCN,
      addSuffix: true,
    });
    // 大约 1 个月前 -> 1 个月前
    // 删除第一个数字之前的文字和空格
    const timeDiff = timeDifference.replace(/^[^\d]*\s*/, "");
    return [startTime, timeDiff];
  }, [date]);

  if (!date) {
    return null;
  }

  return (
    <TooltipProvider delayDuration={10}>
      <Tooltip>
        <TooltipTrigger>
          <Badge
            className="border-none p-0 text-sm font-normal"
            variant="outline"
          >
            {timeDiff}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className="border bg-background text-foreground">
          {format(startTime, "PPPp", { locale: zhCN })}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
