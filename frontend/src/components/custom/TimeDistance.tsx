import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { format, formatDistanceToNow } from "date-fns";
import { zhCN } from "date-fns/locale";
import { useEffect, useState } from "react";

export const TimeDistance = ({
  date,
  className,
}: {
  date?: string;
  className?: string;
}) => {
  const [timeDiff, setTimeDiff] = useState("");
  const [startTime, setStartTime] = useState<Date | null>(null);

  useEffect(() => {
    if (!date) {
      setStartTime(null);
      setTimeDiff("");
      return;
    }

    const time = new Date(date);
    setStartTime(time);
    const updateTimeDiff = () => {
      const timeDifference = formatDistanceToNow(time, {
        locale: zhCN,
        addSuffix: true,
      });
      setTimeDiff(timeDifference.replace(/^[^\d]*\s*/, ""));
    };

    updateTimeDiff();
    const timer = setInterval(updateTimeDiff, 10000);
    return () => clearInterval(timer);
  }, [date]);

  if (!startTime) {
    return null;
  }

  return (
    <TooltipProvider delayDuration={10}>
      <Tooltip>
        <TooltipTrigger className={className}>{timeDiff}</TooltipTrigger>
        <TooltipContent className="border bg-background text-foreground">
          {format(startTime, "PPPp", { locale: zhCN })}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
