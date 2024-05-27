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
      path: "project",
      title: "项目管理",
      children: [
        {
          path: "personal",
          title: "个人项目",
        },
        {
          path: "team",
          title: "团队项目",
        },
      ],
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
      children: [
        {
          path: "public",
          title: "公共镜像",
        },
        {
          path: "personal",
          title: "私有镜像",
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
