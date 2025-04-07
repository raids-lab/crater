import { CopyButton } from "../button/copy-button";

interface CopyableCommandProps {
  label: string;
  command: string;
}

export function CopyableCommand({ label, command }: CopyableCommandProps) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-muted-foreground text-sm font-medium">{label}</span>
      <div className="bg-muted flex flex-row items-center justify-between rounded-md p-2">
        <code className="text-sm break-all">{command}</code>
        <CopyButton content={command} />
      </div>
    </div>
  );
}
