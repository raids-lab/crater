import { CopyButton } from "../button/copy-button";
import { ExternalLink } from "lucide-react";
import TooltipLink from "../label/TooltipLink";

interface CopyableCommandProps {
  label: string;
  command: string;
  isLink?: boolean;
  isSensitive?: boolean;
}

export function CopyableCommand({
  label,
  command,
  isLink = false,
  isSensitive = false,
}: CopyableCommandProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-muted-foreground text-sm font-medium">{label}</span>
      <div className="bg-muted group relative flex flex-row items-center justify-between rounded-md p-2 dark:bg-slate-800/60">
        <div className="mr-2 flex-1 overflow-hidden">
          {isLink ? (
            <div className="flex items-center">
              <code className="mr-2 flex-1 text-sm break-all">{command}</code>
              <TooltipLink
                to={command.startsWith("http") ? command : `http://${command}`}
                name={<ExternalLink className="h-4 w-4" />}
                tooltip="在新标签页中打开"
              />
            </div>
          ) : (
            <code className="text-sm break-all">{command}</code>
          )}
        </div>
        {!isSensitive && <CopyButton content={command} />}
      </div>
    </div>
  );
}
