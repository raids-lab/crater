import { cn } from "@/lib/utils";
import { ImageInfoResponse } from "@/services/api/imagepack";
import { ComboboxItem } from "./Combobox";
import { BoxIcon } from "lucide-react";
import { shortenImageName } from "@/utils/formatter";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { TimeDistance } from "../custom/TimeDistance";

export default function ImageItem({
  item,
}: {
  item: ComboboxItem<ImageInfoResponse>;
}) {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="text-muted-foreground flex w-full items-center gap-3">
            <div
              className={cn(
                "flex size-8 items-center justify-center rounded-full font-normal",
                {
                  "bg-primary/15": item.detail?.isPublic,
                  "bg-purple-500/15": !item.detail?.isPublic,
                },
              )}
            >
              <BoxIcon
                className={cn("size-5", {
                  "text-primary": item.detail?.isPublic,
                  "text-purple-500": !item.detail?.isPublic,
                })}
              />
            </div>
            <div className="flex flex-col items-start gap-0.5">
              {item.detail?.description && (
                <p className="text-foreground">
                  {item.detail?.description ?? item.label}
                </p>
              )}
              <p className="truncate font-mono text-xs" data-description>
                {shortenImageName(item.detail?.imageLink ?? item.value)}
              </p>
            </div>
          </div>
        </TooltipTrigger>
        <TooltipContent className="flex flex-row gap-0.5">
          <p>{item.detail?.userInfo.nickname}</p>创建于
          <TimeDistance date={item.detail?.createdAt} className="text-xs" />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
