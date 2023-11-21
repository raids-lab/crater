type PathInfo = {
  path: string; // router path
  title: string; // title in sidebar
  titleNav?: string; // title in navibar (if not set, use title)
  hiddenNav?: boolean; // if set, this path will not be shown in navibar
  children?: PathInfo[]; // children path info
};

const pathDict: PathInfo[] = [
  {
    path: "portal",
    title: "Portal",
    children: [
      {
        path: "overview",
        title: "概览",
        titleNav: "GPU 集群管理系统",
      },
      {
        path: "jupyter",
        title: "Jupyter 管理",
      },
      {
        path: "job",
        title: "任务管理",
        children: [
          {
            path: "ai",
            title: "AI 训练任务",
            hiddenNav: true,
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
  },
  {
    path: "recommend",
    title: "Recommend",
    children: [
      {
        path: "overview",
        title: "概览",
      },
      {
        path: "job",
        title: "任务管理",
        children: [
          {
            path: "dl",
            title: "深度推荐训练任务",
            hiddenNav: true,
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
  },
  {
    path: "admin",
    title: "Admin",
    children: [
      {
        path: "overview",
        title: "资源监控",
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
            path: "default",
            title: "通用任务",
          },
          {
            path: "ai",
            title: "AI 训练任务",
          },
          {
            path: "dl",
            title: "深度推荐训练任务",
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
  },
];

/**
 *
 * @param path example: ['portal', 'job', 'ai']
 */
export const getBreadcrumbByPath = (
  path: string[],
): { path: string; title: string }[] | null => {
  const result = [];
  let currentPath = pathDict;
  for (let i = 0; i < path.length; i++) {
    const item = currentPath.find((item) => item.path === path[i]);
    if (item) {
      if (item.hiddenNav) {
        return null;
      }
      result.push({ title: item.titleNav ?? item.title, path: item.path });
      currentPath = item.children || [];
    } else {
      break;
    }
  }
  return result;
};

export const getTitleByPath = (path: string[]): string => {
  let currentPath = pathDict;
  for (let i = 0; i < path.length; i++) {
    const item = currentPath.find((item) => item.path === path[i]);
    if (item) {
      if (i === path.length - 1) {
        return item.title;
      }
      currentPath = item.children || [];
    } else {
      break;
    }
  }
  return "";
};
