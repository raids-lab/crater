import {
  ArrowDownIcon,
  ArrowUpIcon,
  CheckCircledIcon,
  CrossCircledIcon,
  StopwatchIcon,
  CircleIcon,
  ClockIcon,
  MinusCircledIcon,
} from "@radix-ui/react-icons";

export const getHeader = (key: string): string => {
  switch (key) {
    case "id":
      return "序号";
    case "title":
      return "任务名称";
    case "taskType":
      return "类型";
    case "gpus":
      return "GPU";
    case "status":
      return "任务状态";
    case "priority":
      return "优先级";
    case "profileStatus":
      return "分析状态";
    case "createdAt":
      return "创建于";
    case "startedAt":
      return "开始于";
    case "finishAt":
      return "完成于";
    default:
      return key;
  }
};

export const priorities = [
  {
    label: "高",
    value: "high",
    icon: ArrowUpIcon,
  },
  {
    label: "低",
    value: "low",
    icon: ArrowDownIcon,
  },
];

export enum JobStatus {
  Initial = 1,
  Created,
  Running,
  Succeeded,
  Failed,
  Preempted,
}

export type StatusValue =
  | "Queueing"
  | "Created"
  //   | "Pending"
  | "Running"
  | "Failed"
  | "Succeeded"
  | "Preempted";

export const statuses: {
  value: StatusValue;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}[] = [
  {
    value: "Queueing",
    label: "检查中",
    icon: CircleIcon,
  },
  {
    value: "Created",
    label: "等待中",
    icon: ClockIcon,
  },
  {
    value: "Running",
    label: "运行中",
    icon: StopwatchIcon,
  },
  {
    value: "Succeeded",
    label: "成功",
    icon: CheckCircledIcon,
  },
  {
    value: "Preempted",
    label: "被抢占",
    icon: MinusCircledIcon,
  },
  {
    value: "Failed",
    label: "失败",
    icon: CrossCircledIcon,
  },
];

export function getStatusValue(status: JobStatus): string {
  switch (status) {
    case JobStatus.Initial:
      return "Created";
    case JobStatus.Created:
      return "Created";
    case JobStatus.Running:
      return "Running";
    case JobStatus.Succeeded:
      return "Succeeded";
    case JobStatus.Failed:
      return "Failed";
    case JobStatus.Preempted:
      return "Preempted";
  }
}

// Profilingstatus
// UnProfiled = 0 // 未分析
// ProfileQueued = 1 // 等待分析
// Profiling = 2 // 正在进行分析
// ProfileFinish = 3 // 分析完成
// ProfileFailed = 4 // 分析失败
export const profilingStatuses = [
  {
    value: "0",
    label: "未分析",
    icon: ClockIcon,
  },
  // {
  //   value: "1",
  //   label: "等待分析",
  //   icon: ClockIcon,
  // },
  {
    value: "2",
    label: "分析中",
    icon: StopwatchIcon,
  },
  {
    value: "3",
    label: "分析完成",
    icon: CheckCircledIcon,
  },
  {
    value: "4",
    label: "分析失败",
    icon: CrossCircledIcon,
  },
];
