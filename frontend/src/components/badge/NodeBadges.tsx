import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { InfoIcon } from "lucide-react";
import { DropdownMenuLabel } from "../ui/dropdown-menu";
import { useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";

const NodeBadges = ({ nodes }: { nodes?: string[] }) => {
  const navigate = useNavigate();

  if (!nodes || nodes.length === 0) {
    return <></>;
  }

  const isSingleNode = nodes.length === 1;
  const handleBadgeClick = () => {
    if (isSingleNode) {
      navigate(`/portal/overview/${nodes[0]}`);
    }
  };

  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge
            variant="secondary"
            className={cn(
              "font-mono font-normal select-none",
              isSingleNode ? "cursor-pointer" : "cursor-help",
            )}
            onClick={handleBadgeClick}
          >
            {isSingleNode ? (
              nodes[0]
            ) : (
              <p>
                {nodes.length}
                <span className="ml-0.5 font-sans">节点</span>
              </p>
            )}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className="p-0">
          {isSingleNode ? (
            <div className="p-2 text-xs">查看节点 {nodes[0]} 详情</div>
          ) : (
            <div className="flex flex-col">
              {nodes
                .sort((a, b) => a.localeCompare(b))
                .map((node, i) => (
                  <div
                    key={node}
                    className={cn("flex flex-col p-1", {
                      "border-t dark:border-slate-700": i > 0,
                    })}
                  >
                    <DropdownMenuLabel className="text-xs">
                      {node}
                    </DropdownMenuLabel>
                    <Button
                      variant="ghost"
                      className="z-10 cursor-pointer justify-start bg-transparent px-2 py-1"
                      onClick={() => navigate(`/portal/overview/${node}`)}
                    >
                      <InfoIcon className="mr-2 h-4 w-4" />
                      <span className="truncate font-normal">节点详情</span>
                    </Button>
                  </div>
                ))}
            </div>
          )}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

export default NodeBadges;
