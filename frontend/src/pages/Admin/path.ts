import { PathInfo } from "@/utils/title";

export const adminPath: PathInfo = {
  path: "admin",
  title: "Admin",
  children: [
    {
      path: "overview",
      title: "节点管理",
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
          title: "Jupyter 任务",
        },
        {
          path: "ai",
          title: "AI 训练任务",
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
          title: "公开数据集",
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
