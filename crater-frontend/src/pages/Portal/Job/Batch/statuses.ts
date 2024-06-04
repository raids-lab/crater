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
    case "gpus":
      return "GPU";
    case "status":
      return "作业状态";
    case "priority":
      return "优先级";
    case "profileStatus":
      return "分析状态";
    case "createdAt":
      return "创建于";
    case "startedAt":
      return "开始于";
    case "completedAt":
      return "完成于";
    default:
      return key;
  }
};
