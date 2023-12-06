import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { getDateDiff } from "@/utils/formatter";

export const TableDate = ({ date }: { date: string }) => {
  const createdAt = new Date(date);
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>{getDateDiff(date)}</TooltipTrigger>
        <TooltipContent>
          <p>
            {createdAt.toLocaleString("zh-CN", {
              timeZone: "Asia/Shanghai",
            })}
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
