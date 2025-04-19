import { JobType } from "@/services/api/vcjob";
import { PhaseBadge } from "./PhaseBadge";

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
  description: string;
} => {
  switch (phase) {
    case JobType.Custom:
      return {
        label: "Custom",
        color: "text-highlight-purple bg-highlight-purple/10",
        description: "自定义作业",
      };
    case JobType.Jupyter:
      return {
        label: "Jupyter",
        color: "text-highlight-amber bg-highlight-amber/10",
        description: "Jupyter 交互式作业",
      };
    case JobType.Tensorflow:
      return {
        label: "Tensorflow",
        color: "text-highlight-cyan bg-highlight-cyan/10",
        description: "TensorFlow PS 作业",
      };
    case JobType.Pytorch:
      return {
        label: "Pytorch",
        color: "text-highlight-rose bg-highlight-rose/10",
        description: "Pytorch DDP 作业",
      };
    case JobType.WebIDE:
      return {
        label: "WebIDE",
        color: "text-highlight-lime bg-highlight-lime/10",
        description: "WebIDE 交互式作业",
      };
    default:
      return {
        label: "Custom",
        color: "text-highlight-purple bg-highlight-purple/10",
        description: "自定义作业",
      };
  }
};

const JobTypeLabel = ({ jobType }: { jobType: JobType }) => {
  return <PhaseBadge phase={jobType} getPhaseLabel={getJobTypeLabel} />;
};

export default JobTypeLabel;
