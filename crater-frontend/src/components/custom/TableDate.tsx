import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { getDateDiff } from "@/utils/formatter";
import { Badge } from "../ui/badge";

export const TableDate = ({ date }: { date: string }) => {
  const createdAt = new Date(date);
  return (
    <TooltipProvider delayDuration={10}>
      <Tooltip>
        <TooltipTrigger>
          <Badge
            className="border-none p-0 text-sm font-normal"
            variant="outline"
          >
            {getDateDiff(date)}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className="border bg-background text-foreground">
          {createdAt.toLocaleString("zh-CN", {
            timeZone: "Asia/Shanghai",
          })}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
