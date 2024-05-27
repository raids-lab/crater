import { JobPhase } from "@/services/api/vcjob";
import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";

export const jobPhases = [
  {
    value: "Pending",
    label: "等待中",
  },
  {
    value: "Aborting",
    label: "即将中止",
  },
  {
    value: "Aborted",
    label: "已中止",
  },
  {
    value: "Running",
    label: "运行中",
  },
  {
    value: "Restarting",
    label: "重启中",
  },
  {
    value: "Completing",
    label: "即将完成",
  },
  {
    value: "Completed",
    label: "已完成",
  },
  {
    value: "Terminating",
    label: "即将终止",
  },
  {
    value: "Terminated",
    label: "已终止",
  },
  {
    value: "Failed",
    label: "失败",
  },
];

export const getJobPhaseLabel = (
  phase: JobPhase,
): {
  label: string;
  color: string;
} => {
  switch (phase) {
    case JobPhase.Pending:
      return {
        label: "等待中",
        color: "text-purple-500 border-purple-500 bg-purple-500/10",
      };
    case JobPhase.Aborting:
      return {
        label: "即将中止",
        color: "text-pink-500 border-pink-500 bg-pink-500/10",
      };
    case JobPhase.Aborted:
      return {
        label: "已中止",
        color: "text-pink-500 border-pink-500 bg-pink-500/10",
      };
    case JobPhase.Running:
      return {
        label: "运行中",
        color: "text-blue-500 border-blue-500 bg-blue-500/10",
      };
    case JobPhase.Restarting:
      return {
        label: "重启中",
        color: "text-rose-500 border-rose-500 bg-rose-500/10",
      };
    case JobPhase.Completing:
      return {
        label: "即将完成",
        color: "text-emerald-500 border-emerald-500 bg-emerald-500/10",
      };
    case JobPhase.Completed:
      return {
        label: "已完成",
        color: "text-emerald-500 border-emerald-500 bg-emerald-500/10",
      };
    case JobPhase.Terminating:
      return {
        label: "即将终止",
        color: "text-orange-500 border-orange-500 bg-orange-500/10",
      };
    case JobPhase.Terminated:
      return {
        label: "已终止",
        color: "text-orange-500 border-orange-500 bg-orange-500/10",
      };
    case JobPhase.Failed:
      return {
        label: "失败",
        color: "text-red-500 border-red-500 bg-red-500/10",
      };
    default:
      return {
        label: "未知",
        color: "text-slate-500 border-slate-500 bg-slate-500/10",
      };
  }
};

const JobPhaseLabel = ({ jobPhase }: { jobPhase: JobPhase }) => {
  const data = getJobPhaseLabel(jobPhase);

  return (
    <Badge className={cn("", data.color)} variant="outline">
      <div className="">{data.label}</div>
    </Badge>
  );
};

export default JobPhaseLabel;
