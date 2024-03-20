import { PathInfo } from "@/utils/title";

export const adminPath: PathInfo = {
  path: "admin",
  title: "Admin",
  children: [
    {
      path: "cluster",
      title: "集群管理",
      children: [
        {
          path: "node",
          title: "计算节点",
        },
        {
          path: "gpu",
          title: "加速资源",
        },
      ],
    },
    {
      path: "user",
      title: "账户管理",
      children: [
        {
          path: "personal",
          title: "个人用户",
        },
        {
          path: "group",
          title: "组用户",
        },
      ],
    },
    {
      path: "job",
      title: "任务管理",
      children: [
        {
          path: "jupyter",
          title: "交互式任务",
        },
        {
          path: "ai",
          title: "离线任务",
        },
      ],
    },
    {
      path: "image",
      title: "镜像管理",
      children: [
        {
          path: "list",
          title: "镜像列表",
        },
      ],
    },
    {
      path: "data",
      title: "数据管理",
      children: [
        {
          path: "dataset",
          title: "系统文件",
        },
      ],
    },
    {
      path: "docs",
      title: "文档编辑",
    },
    {
      path: "feedback",
      title: "问题反馈",
    },
  ],
};
