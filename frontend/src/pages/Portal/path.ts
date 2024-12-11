import { PathInfo } from "@/utils/title";

export const craterPath: PathInfo = {
  path: "portal",
  title: "Portal",
  isEmpty: true,
  children: [
    {
      path: "overview",
      title: "平台概览",
      titleNav: "异构云资源混合调度与运维平台",
    },
    {
      path: "job",
      title: "我的作业",
      isEmpty: true,
      children: [
        {
          path: "batch",
          title: "批处理作业",
          children: [
            {
              path: "new-aijobs",
              title: "新建自定义作业",
            },
            {
              path: "new-spjobs",
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
              path: "new-jupyter-vcjobs",
              title: "新建 Jupyter Lab",
            },
            {
              path: "new-jupyter-aijobs",
              title: "新建 Jupyter Lab",
            },
          ],
        },
      ],
    },
    {
      path: "service",
      title: "我的服务",
      isEmpty: true,
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
      title: "我的镜像",
      isEmpty: true,
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
      isEmpty: true,
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
      isEmpty: true,
      children: [
        {
          path: "member",
          title: "成员管理",
        },
      ],
    },
    {
      path: "setting",
      title: "设置",
      children: [
        {
          path: "user",
          title: "个人主页",
        },
        {
          path: "platform",
          title: "平台设置",
        },
      ],
    },
    {
      path: "feedback",
      title: "帮助与反馈",
    },
  ],
};
