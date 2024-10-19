import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { jobPhases } from "@/components/label/JobPhaseLabel";
import { jobTypes } from "@/components/custom/JobTypeLabel";

export const getHeader = (key: string): string => {
  switch (key) {
    case "id":
      return "序号";
    case "name":
      return "名称";
    case "jobType":
      return "类型";
    case "owner":
      return "用户";
    case "status":
      return "状态";
    case "nodes":
      return "节点";
    case "resources":
      return "资源";
    case "priority":
      return "优先级";
    case "profileStatus":
      return "分析状态";
    case "createdAt":
      return "创建于";
    case "startedAt":
      return "开始于";
    case "completedAt":
      return "结束于";
    default:
      return key;
  }
};

export const jobToolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索作业名称",
    key: "name",
  },
  filterOptions: [
    {
      key: "jobType",
      title: "类型",
      option: jobTypes,
    },
    {
      key: "status",
      title: "状态",
      option: jobPhases,
    },
  ],
  getHeader: getHeader,
};
