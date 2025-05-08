import { PathInfo } from "@/utils/title";

export const craterPath: PathInfo = {
  path: "portal",
  titleKey: "navigation.portal",
  isEmpty: true,
  children: [
    {
      path: "overview",
      titleKey: "navigation.platformOverview",
      titleNavKey: "navigation.platformFullName",
    },
    {
      path: "monitor",
      titleKey: "navigation.clusterMonitoring",
      isEmpty: true,
      children: [
        {
          path: "gpu",
          titleKey: "navigation.gpuMonitoring",
        },
        {
          path: "node",
          titleKey: "navigation.freeResources",
        },
        {
          path: "network",
          titleKey: "navigation.networkMonitoring",
        },
      ],
    },
    {
      path: "job",
      titleKey: "navigation.myJobs",
      isEmpty: true,
      children: [
        {
          path: "batch",
          titleKey: "navigation.customJobs",
          children: [
            {
              path: "new-vcjobs",
              titleKey: "navigation.createCustomJob",
            },
            {
              path: "new-aijobs",
              titleKey: "navigation.createCustomJob",
            },
            {
              path: "new-spjobs",
              titleKey: "navigation.createCustomJob",
            },
            {
              path: "new-tensorflow",
              titleKey: "navigation.createTensorflowJob",
            },
            {
              path: "new-pytorch",
              titleKey: "navigation.createPytorchJob",
            },
          ],
        },
        {
          path: "inter",
          titleKey: "navigation.jupyterLab",
          children: [
            {
              path: "new-jupyter-vcjobs",
              titleKey: "navigation.createJupyterLab",
            },
            {
              path: "new-jupyter-aijobs",
              titleKey: "navigation.createJupyterLab",
            },
          ],
        },
      ],
    },
    {
      path: "modal",
      titleKey: "navigation.jobTemplates",
      isEmpty: true,
    },
    {
      path: "image",
      titleKey: "navigation.myImages",
      isEmpty: true,
      children: [
        {
          path: "createimage",
          titleKey: "navigation.imageCreation",
        },
        {
          path: "uploadimage",
          titleKey: "navigation.imageList",
        },
      ],
    },
    {
      path: "data",
      titleKey: "navigation.dataManagement",
      isEmpty: true,
      children: [
        {
          path: "filesystem",
          titleKey: "navigation.fileSystem",
        },
        {
          path: "dataset",
          titleKey: "navigation.datasets",
        },
        {
          path: "model",
          titleKey: "navigation.models",
        },
        {
          path: "sharefile",
          titleKey: "navigation.sharedFiles",
        },
      ],
    },
    {
      path: "account",
      titleKey: "navigation.accountManagement",
      isEmpty: true,
      children: [
        {
          path: "member",
          titleKey: "navigation.memberManagement",
        },
      ],
    },
    {
      path: "setting",
      titleKey: "navigation.settings",
      children: [
        {
          path: "user",
          titleKey: "navigation.userSettings",
        },
        {
          path: "platform",
          titleKey: "navigation.platformSettings",
        },
      ],
    },
    {
      path: "feedback",
      titleKey: "navigation.helpAndFeedback",
    },
  ],
};
