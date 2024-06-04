import { PathInfo } from "@/utils/title";

export const recommendPath: PathInfo = {
  path: "recommend",
  title: "Recommend",
  children: [
    {
      path: "overview",
      title: "集群概览",
    },
    {
      path: "job",
      title: "作业管理",
      children: [
        {
          path: "dl",
          title: "深度推荐训练作业",
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
          path: "dataset",
          title: "数据集",
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
