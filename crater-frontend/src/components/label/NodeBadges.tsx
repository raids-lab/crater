import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { GaugeIcon } from "lucide-react";
import { Button } from "../ui/button";

const GRAFANA_NODE = import.meta.env.VITE_GRAFANA_NODE;

const NodeBadges = ({ nodes }: { nodes?: string[] }) => {
  if (!nodes || nodes.length === 0) {
    return <></>;
  }

  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger disabled>
          <Badge
            variant="secondary"
            className="cursor-pointer select-none font-mono font-normal"
          >
            {nodes.length > 1 ? (
              <p>
                {nodes.length}
                <span className="ml-0.5 font-sans">节点</span>
              </p>
            ) : (
              nodes[0]
            )}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className="border bg-background p-0 text-foreground">
          <div className="flex flex-col gap-1 p-1">
            {nodes
              .sort((a, b) => a.localeCompare(b))
              .map((node) => (
                <Button
                  variant="ghost"
                  key={node}
                  title="查看节点资源监控"
                  className="flex h-8 w-full flex-row justify-start px-2 pb-1 font-normal"
                  onClick={() => {
                    window.open(
                      `${GRAFANA_NODE}?orgId=1&refresh=5m&var-datasource=prometheus&var-instance=${node}`,
                    );
                  }}
                >
                  <GaugeIcon className="h-4 w-4" />
                  <div className="font-mono">{node}</div>
                </Button>
              ))}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

export default NodeBadges;
