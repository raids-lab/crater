"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Check, Copy } from "lucide-react";
import { toast } from "sonner";
import { logger } from "@/utils/loglevel";

interface CopyableCommandProps {
  label: string;
  command: string;
}

export function CopyableCommand({ label, command }: CopyableCommandProps) {
  const [isCopied, setIsCopied] = useState(false);

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(command);
      setIsCopied(true);
      toast.success("已复制到剪贴板");
      setTimeout(() => setIsCopied(false), 2000);
    } catch (err) {
      logger.error(err);
      toast.error("复制失败，请手动复制");
    }
  };

  return (
    <div className="bg-muted flex flex-col space-y-1 rounded-md p-2">
      <div className="flex items-center justify-between">
        <span className="text-muted-foreground font-medium">{label}:</span>
        <Button
          size="icon"
          variant="ghost"
          className="cursor-pointer"
          onClick={copyToClipboard}
        >
          {isCopied ? (
            <Check className="h-4 w-4 text-green-500" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </Button>
      </div>
      <code className="text-sm break-all">{command}</code>
    </div>
  );
}
