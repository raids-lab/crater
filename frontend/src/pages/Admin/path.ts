import { PathInfo } from "@/utils/title";

export const adminPath: PathInfo = {
  path: "admin",
  title: "Admin",
  isEmpty: true,
  children: [
    {
      path: "cluster",
      title: "集群管理",
      isEmpty: true,
      children: [
        {
          path: "node",
          title: "节点信息",
        },
        {
          path: "label",
          title: "节点标签",
        },
        {
          path: "resource",
          title: "资源列表",
        },
      ],
    },
    {
      path: "account",
      title: "账户管理",
    },
    {
      path: "user",
      title: "用户管理",
    },
    {
      path: "job",
      title: "作业管理",
    },
    {
      path: "image",
      title: "镜像管理",
      isEmpty: true,
      children: [
        {
          path: "create",
          title: "镜像制作",
        },
        {
          path: "upload",
          title: "镜像列表",
        },
      ],
    },
    {
      path: "data",
      title: "数据管理",
      isEmpty: true,
      children: [
        {
          path: "user",
          title: "用户文件",
        },
        {
          path: "account",
          title: "账户文件",
        },
        {
          path: "dataset",
          title: "数据集",
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
