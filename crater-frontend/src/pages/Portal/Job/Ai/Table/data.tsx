import { DataTableToolbarConfig } from "@/components/DataTable/DataTableToolbar";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  CheckCircledIcon,
  CircleIcon,
  CrossCircledIcon,
  QuestionMarkCircledIcon,
  StopwatchIcon,
} from "@radix-ui/react-icons";

export const priorities = [
  {
    label: "高优先级",
    value: "high",
    icon: ArrowUpIcon,
  },
  {
    label: "低优先级",
    value: "low",
    icon: ArrowDownIcon,
  },
];

export const statuses = [
  {
    value: "Queueing",
    label: "Queueing",
    icon: QuestionMarkCircledIcon,
  },
  {
    value: "Pending",
    label: "Pending",
    icon: CircleIcon,
  },
  {
    value: "Running",
    label: "Running",
    icon: StopwatchIcon,
  },
  {
    value: "Failed",
    label: "Failed",
    icon: CrossCircledIcon,
  },
  {
    value: "Succeeded",
    label: "Succeeded",
    icon: CheckCircledIcon,
  },
  {
    value: "Suspended",
    label: "Suspended",
    icon: CrossCircledIcon,
  },
];

export const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索任务标题",
    key: "title",
  },
  filterOptions: [
    {
      key: "status",
      title: "状态",
      option: statuses,
    },
    {
      key: "priority",
      title: "优先级",
      option: priorities,
    },
  ],
};
