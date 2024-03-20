import { PathInfo } from "@/utils/title";

export const craterPath: PathInfo = {
  path: "portal",
  title: "Portal",
  children: [
    {
      path: "overview",
      title: "概览",
      titleNav: "GPU 集群管理系统",
    },
    {
      path: "job",
      title: "任务管理",
      children: [
        {
          path: "ai",
          title: "离线任务",
        },
        {
          path: "jupyter",
          title: "交互式任务",
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
        {
          path: "make",
          title: "镜像制作",
        },
      ],
    },
    {
      path: "data",
      title: "数据管理",
      children: [
        {
          path: "share",
          title: "共享文件",
        },
        {
          path: "other",
          title: "我的文件",
        },
      ],
    },
    {
      path: "docs",
      title: "使用文档",
    },
    {
      path: "feedback",
      title: "问题反馈",
    },
  ],
};
