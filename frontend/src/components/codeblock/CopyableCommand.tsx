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
      setTimeout(() => setIsCopied(false), 2000);
    } catch (err) {
      logger.error(err);
      toast.error("Failed to copy text:");
    }
  };

  return (
    <div className="flex·flex-col·space-y-1·rounded-md·bg-muted·p-2">
      <div className="flex items-center justify-between">
        <span className="font-medium">{label}:</span>
        <Button size="icon" variant="ghost" onClick={copyToClipboard}>
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
