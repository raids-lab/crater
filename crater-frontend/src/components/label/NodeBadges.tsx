import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import GpuIcon from "@/components/icon/GpuIcon";
import { Button } from "@/components/ui/button";
import { CpuIcon } from "lucide-react";
import { DropdownMenuLabel } from "../ui/dropdown-menu";
import { InfoCircledIcon } from "@radix-ui/react-icons";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

const GRAFANA_NODE = import.meta.env.VITE_GRAFANA_NODE;

const NodeBadges = ({ nodes }: { nodes?: string[] }) => {
  const navigate = useNavigate();

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
          <div className="flex flex-row">
            {nodes
              .sort((a, b) => a.localeCompare(b))
              .map((node, i) => (
                <div
                  className={cn("flex flex-col p-1", {
                    "border-l": i > 0,
                  })}
                >
                  <DropdownMenuLabel
                    key={node}
                    className="text-xs text-muted-foreground"
                  >
                    {node}
                  </DropdownMenuLabel>
                  {/* 按钮 */}
                  <Button
                    variant="ghost"
                    className="justify-start px-2 py-1"
                    onClick={() => navigate(`/portal/overview/${node}`)}
                  >
                    <InfoCircledIcon className="text-emerald-600 dark:text-emerald-500" />
                    <span className="truncate font-normal">节点详情</span>
                  </Button>
                  <Button
                    variant="ghost"
                    className="justify-start px-2 py-1"
                    onClick={() => {
                      window.open(
                        `${GRAFANA_NODE}?orgId=1&refresh=5m&var-datasource=prometheus&var-node=${node}`,
                      );
                    }}
                  >
                    <CpuIcon className="text-purple-600 dark:text-purple-500" />
                    <span className="truncate font-normal">节点监控</span>
                  </Button>
                  <Button
                    variant="ghost"
                    className="justify-start px-2 py-1"
                    onClick={() =>
                      toast.warning("TODO(zhangry): add GPU monitoring")
                    }
                  >
                    <GpuIcon className="text-orange-600 dark:text-orange-500" />
                    <span className="truncate font-normal">加速卡监控</span>
                  </Button>
                </div>
              ))}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

export default NodeBadges;
