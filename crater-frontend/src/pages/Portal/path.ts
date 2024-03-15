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
          title: "AI 训练任务",
        },
        {
          path: "jupyter",
          title: "Jupyter 任务",
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
          title: "共享空间",
        },
        {
          path: "code",
          title: "代码",
        },
        {
          path: "other",
          title: "其他",
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
