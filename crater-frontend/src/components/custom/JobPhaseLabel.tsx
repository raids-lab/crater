import { JobPhase } from "@/services/api/vcjob";
import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export const jobPhases = [
  {
    value: "Pending",
    label: "等待中",
    color: "#a855f7",
  },
  {
    value: "Aborting",
    label: "即将中止",
    color: "#ec4899",
  },
  {
    value: "Aborted",
    label: "已中止",
    color: "#ec4899",
  },
  {
    value: "Running",
    label: "运行中",
    color: "#3b82f6",
  },
  {
    value: "Restarting",
    label: "重启中",
    color: "#f43f5e",
  },
  {
    value: "Completing",
    label: "即将完成",
    color: "#10b981",
  },
  {
    value: "Completed",
    label: "已完成",
    color: "#10b981",
  },
  {
    value: "Terminating",
    label: "即将终止",
    color: "#f97316",
  },
  {
    value: "Terminated",
    label: "已终止",
    color: "#f97316",
  },
  {
    value: "Deleted",
    label: "已停止",
    color: "#f97316",
  },
  {
    value: "Freed",
    label: "已释放",
    color: "#f97316",
  },
  {
    value: "Failed",
    label: "失败",
    color: "#ef4444",
  },
];

export const getJobPhaseLabel = (
  phase: JobPhase,
): {
  label: string;
  color: string;
  description: string;
} => {
  switch (phase) {
    case JobPhase.Pending:
      return {
        label: "等待中",
        color: "text-purple-500 border-purple-500 bg-purple-500/10",
        description:
          "作业正在排队等待执行，如果等待时间过长，可能是集群资源不足或账户配额已达限制",
      };
    case JobPhase.Aborting:
      return {
        label: "即将中止",
        color: "text-pink-500 border-pink-500 bg-pink-500/10",
        description: "作业即将被中止",
      };
    case JobPhase.Aborted:
      return {
        label: "已中止",
        color: "text-pink-500 border-pink-500 bg-pink-500/10",
        description: "作业已被中止",
      };
    case JobPhase.Running:
      return {
        label: "运行中",
        color: "text-blue-500 border-blue-500 bg-blue-500/10",
        description: "作业正在运行，您可以点击作业名称查看详情",
      };
    case JobPhase.Restarting:
      return {
        label: "重启中",
        color: "text-rose-500 border-rose-500 bg-rose-500/10",
        description: "作业正在重启",
      };
    case JobPhase.Completing:
      return {
        label: "即将完成",
        color: "text-emerald-500 border-emerald-500 bg-emerald-500/10",
        description: "作业即将完成",
      };
    case JobPhase.Completed:
      return {
        label: "已完成",
        color: "text-emerald-500 border-emerald-500 bg-emerald-500/10",
        description: "作业已完成",
      };
    case JobPhase.Terminating:
      return {
        label: "即将终止",
        color: "text-orange-500 border-orange-500 bg-orange-500/10",
        description: "作业即将终止",
      };
    case JobPhase.Terminated:
      return {
        label: "已终止",
        color: "text-orange-500 border-orange-500 bg-orange-500/10",
        description: "作业已终止",
      };
    case JobPhase.Failed:
      return {
        label: "失败",
        color: "text-red-500 border-red-500 bg-red-500/10",
        description: "作业失败",
      };
    case JobPhase.Deleted:
      return {
        label: "已停止",
        color: "text-red-500 border-red-500 bg-red-500/10",
        description: "作业被手动停止，相关资源已删除，仅保留元数据",
      };
    case JobPhase.Freed:
      return {
        label: "已释放",
        color: "text-red-500 border-red-500 bg-red-500/10",
        description: "由于您的作业长期占用资源但未使用，已自动释放",
      };
    default:
      return {
        label: "未知",
        color: "text-slate-500 border-slate-500 bg-slate-500/10",
        description: "作业状态未知",
      };
  }
};

const JobPhaseLabel = ({ jobPhase }: { jobPhase: JobPhase }) => {
  const data = getJobPhaseLabel(jobPhase);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger disabled>
          <Badge className={cn("", data.color)} variant="outline">
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

export default JobPhaseLabel;
