import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export const podPhases = [
  {
    value: "Pending",
    label: "等待中",
    color: "text-purple-500 border-purple-500 bg-purple-500/10",
    description: "Pod 正在等待被调度或容器镜像正在下载",
  },
  {
    value: "Running",
    label: "运行中",
    color: "text-blue-500 border-blue-500 bg-blue-500/10",
    description: "Pod 已被调度到节点上且至少一个容器正在运行",
  },
  {
    value: "Succeeded",
    label: "已完成",
    color: "text-emerald-500 border-emerald-500 bg-emerald-500/10",
    description: "Pod 中的所有容器已成功终止且不会再启动",
  },
  {
    value: "Failed",
    label: "已失败",
    color: "text-red-500 border-red-500 bg-red-500/10",
    description: "Pod 中的所有容器已终止且至少一个容器因错误终止",
  },
  {
    value: "Unknown",
    label: "未知",
    color: "text-slate-500 border-slate-500 bg-slate-500/10",
    description: "由于某些原因无法获取 Pod 的状态信息",
  },
];

const getPodPhaseLabel = (
  phase: string,
): {
  label: string;
  color: string;
  description: string;
} => {
  const foundPhase = podPhases.find((p) => p.value === phase);
  if (foundPhase) {
    return foundPhase;
  } else {
    return {
      label: "未知",
      color: "#808080",
      description: "由于某些原因无法获取 Pod 的状态信息",
    };
  }
};

const PodPhaseLabel = ({ podPhase }: { podPhase: string }) => {
  const data = getPodPhaseLabel(podPhase);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger disabled>
          <Badge className={cn("cursor-pointer", data.color)} variant="outline">
            <div className="">{data.label}</div>
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          <p>{data.description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

export default PodPhaseLabel;
