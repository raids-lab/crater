import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { Copy } from "lucide-react";

interface TooltipCopyProps {
  name: string;
  className?: string;
  copyMessage?: string;
  showIcon?: boolean;
}

export default function TooltipCopy({
  name,
  className,
  copyMessage,
  showIcon = false,
}: TooltipCopyProps) {
  const { handleCopy } = useCopyToClipboard({
    text: name,
    copyMessage: copyMessage || `已复制 "${name}" 到剪贴板`,
  });

  return (
    <TooltipProvider delayDuration={50}>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            onClick={handleCopy}
            className={cn(
              "hover:text-primary inline-flex cursor-pointer items-center gap-1 font-normal",
              className,
            )}
          >
            {name}
            {showIcon && <Copy className="h-3.5 w-3.5" />}
          </button>
        </TooltipTrigger>
        <TooltipContent>点击以复制</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
