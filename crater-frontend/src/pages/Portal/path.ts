import { PathInfo } from "@/utils/title";

export const craterPath: PathInfo = {
  path: "portal",
  title: "Portal",
  children: [
    {
      path: "overview",
      title: "集群概览",
      titleNav: "GPU 集群管理系统",
    },
    {
      path: "job",
      title: "作业管理",
      children: [
        {
          path: "batch",
          title: "批处理作业",
          children: [
            {
              path: "new-custom",
              title: "新建自定义作业",
            },
            {
              path: "new-tensorflow",
              title: "新建 Tensorflow 作业",
            },
            {
              path: "new-pytorch",
              title: "新建 Pytorch 作业",
            },
          ],
        },
        {
          path: "inter",
          title: "交互式作业",
          children: [
            {
              path: "new-jupyter",
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
          path: "createimage",
          title: "镜像制作",
        },
        {
          path: "uploadimage",
          title: "镜像列表",
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
      path: "account",
      title: "账户管理",
      children: [
        {
          path: "member",
          title: "成员管理",
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
