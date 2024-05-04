import { JobPhase } from "@/services/api/jupyterTask";
import { cn } from "@/lib/utils";

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
        color: "bg-purple-500 hover:bg-purple-400",
      };
    case JobPhase.Aborting:
      return {
        label: "即将中止",
        color: "bg-pink-500 hover:bg-pink-400",
      };
    case JobPhase.Aborted:
      return {
        label: "已中止",
        color: "bg-pink-500 hover:bg-pink-400",
      };
    case JobPhase.Running:
      return {
        label: "运行中",
        color: "bg-sky-500 hover:bg-sky-400",
      };
    case JobPhase.Restarting:
      return {
        label: "重启中",
        color: "bg-rose-500 hover:bg-rose-200",
      };
    case JobPhase.Completing:
      return {
        label: "即将完成",
        color: "bg-emerald-500 hover:bg-emerald-400",
      };
    case JobPhase.Completed:
      return {
        label: "已完成",
        color: "bg-emerald-500 hover:bg-emerald-400",
      };
    case JobPhase.Terminating:
      return {
        label: "即将终止",
        color: "bg-orange-500 hover:bg-orange-400",
      };
    case JobPhase.Terminated:
      return {
        label: "已终止",
        color: "bg-orange-500 hover:bg-orange-400",
      };
    case JobPhase.Failed:
      return {
        label: "失败",
        color: "bg-red-500 hover:bg-red-400",
      };
    default:
      return {
        label: "未知",
        color: "bg-slate-500 hover:bg-slate-400",
      };
  }
};

const JobPhaseLabel = ({ jobPhase }: { jobPhase: JobPhase }) => {
  const data = getJobPhaseLabel(jobPhase);

  return (
    <div className="flex flex-row items-center justify-start">
      <div className={cn("flex h-2.5 w-2.5 rounded-full", data.color)}></div>
      <div className="ml-1.5">{data.label}</div>
    </div>
  );
};

export default JobPhaseLabel;
