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
      title: "作业管理",
      children: [
        {
          path: "ai",
          title: "批处理作业",
        },
        {
          path: "jupyter",
          title: "交互式作业",
          children: [
            {
              path: "new",
              title: "新建作业",
            },
          ],
        },
      ],
    },
    {
      path: "service",
      title: "服务管理",
      children: [
        {
          path: "microservice",
          title: "微服务",
        },
        {
          path: "serverless",
          title: "无服务",
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
          path: "filesystem",
          title: "文件系统",
        },
        {
          path: "dataset",
          title: "数据集",
        },
      ],
    },
    {
      path: "project",
      title: "项目管理",
      children: [
        {
          path: "member",
          title: "项目成员",
        },
        {
          path: "config",
          title: "项目配置",
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
