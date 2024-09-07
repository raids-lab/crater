import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";
import { JobType } from "@/services/api/vcjob";

export const jobTypes = [
  {
    value: "custom",
    label: "Custom",
  },
  {
    value: "jupyter",
    label: "Jupyter",
  },
  {
    value: "tensorflow",
    label: "Tensorflow",
  },
  {
    value: "pytorch",
    label: "Pytorch",
  },
  {
    value: "webide",
    label: "Webide",
  },
];

const getJobTypeLabel = (
  phase: JobType,
): {
  label: string;
  color: string;
} => {
  switch (phase) {
    case JobType.Custom:
      return {
        label: "Custom",
        color: "text-purple-600 bg-purple-500/10",
      };
    case JobType.Jupyter:
      return {
        label: "Jupyter",
        color: "text-amber-600 bg-amber-500/10",
      };
    case JobType.Tensorflow:
      return {
        label: "Tensorflow",
        color: "text-cyan-600 bg-cyan-500/10",
      };
    case JobType.Pytorch:
      return {
        label: "Pytorch",
        color: "text-rose-600 bg-rose-500/10",
      };
    case JobType.WebIDE:
      return {
        label: "WebIDE",
        color: "text-lime-600 bg-lime-500/10",
      };
    default:
      return {
        label: "未知",
        color: "text-slate-500 bg-slate-500/10",
      };
  }
};

const JobTypeLabel = ({ jobType }: { jobType: JobType }) => {
  const data = getJobTypeLabel(jobType);

  return (
    <Badge className={cn("border-none", data.color)} variant="outline">
      <div className="">{data.label}</div>
    </Badge>
  );
};

export default JobTypeLabel;
