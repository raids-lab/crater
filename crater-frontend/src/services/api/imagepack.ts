import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import {
  CheckCircledIcon,
  CircleIcon,
  ClockIcon,
  CrossCircledIcon,
  StopwatchIcon,
} from "@radix-ui/react-icons";

export type ImagePackListResponse = {
  imagepacklist: ImagePackInfoResponse[];
  totalsize: number;
};

type ImagePackInfoResponse = {
  ID: number;
  // name: string;
  imagelink: string;
  status: string;
  createdAt: string;
  nametag: string;
  creatername: string;
  imagetype: number;
  description: string;
  alias: string;
  tasktype: number;
  ispublic: boolean;
  params: {
    Convs: number;
    Activations: number;
    Denses: number;
    Others: number;
    GFLOPs: number;
    BatchSize: number;
    Params: number;
    ModelSize: number;
  };
};

export type ImagePackLogResponse = {
  content: string;
};

export type ImagePackInfo = {
  id: number;
  nametag: string;
  link: string;
  username: string;
  status: string;
  createdAt: string;
  imagetype: number;
  tasktype: number;
  ispublic: boolean;
  params: {
    Convs: number;
    Activations: number;
    Denses: number;
    Others: number;
    GFLOPs: number;
    BatchSize: number;
    Params: number;
    ModelSize: number;
  };
};

export const getHeader = (key: string): string => {
  switch (key) {
    case "nametag":
      return "名称";
    case "link":
      return "镜像地址";
    case "username":
      return "提交者";
    case "status":
      return "状态";
    case "createdAt":
      return "创建时间";
    case "tasktype":
      return "任务类型";
    default:
      return key;
  }
};

export type ImagePackStatusValue =
  | "Initial"
  | "Pending"
  | "Running"
  | "Finished"
  | "Failed"
  | "";

export const imagepackStatuses: {
  value: ImagePackStatusValue;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}[] = [
  {
    value: "Initial",
    label: "检查中",
    icon: CircleIcon,
  },
  {
    value: "Pending",
    label: "等待中",
    icon: ClockIcon,
  },
  {
    value: "Running",
    label: "运行中",
    icon: StopwatchIcon,
  },
  {
    value: "Finished",
    label: "成功",
    icon: CheckCircledIcon,
  },
  {
    value: "Failed",
    label: "失败",
    icon: CrossCircledIcon,
  },
];

export type imagepackTaskTypeValue = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8;

export const imagepackTaskType: {
  value: imagepackTaskTypeValue;
  label: string;
}[] = [
  {
    value: 1,
    label: "jupyter",
  },
  {
    value: 2,
    label: "webide",
  },
  {
    value: 3,
    label: "tensorflow",
  },
  {
    value: 4,
    label: "pytorch",
  },
  {
    value: 5,
    label: "ray",
  },
  {
    value: 6,
    label: "deepspeed",
  },
  {
    value: 7,
    label: "openmpi",
  },
  {
    value: 8,
    label: "training",
  },
];

export interface ImagePackCreate {
  gitRepository: string;
  accessToken: string;
  dockerfile: string;
  registryServer: string;
  registryUser: string;
  registryPass: string;
  registryProject: string;
  imageName: string;
  imageTag: string;
  needProfile: boolean;
  alias: string;
  description: string;
  taskType: number;
}

export interface ImagePackUpload {
  imageLink: string;
  imageName: string;
  imageTag: string;
  alias: string;
  description: string;
  taskType: number;
}

export const ImageTaskType = {
  JupyterTask: 1, // Jupyter交互式任务
  WebIDETask: 2, // Web IDE任务
  TensorflowTask: 3, // Tensorflow任务
  PytorchTask: 4, // Pytorch任务
  RayTask: 5, // Ray任务
  DeepSpeedTask: 6, // DeepSpeed任务
  OpenMPITask: 7, // OpenMPI任务
  UserDefineTask: 8, // 用户自定义任务
};

export const imageLinkRegex =
  /^[a-zA-Z0-9.-]+(\/[a-zA-Z0-9.-]+)+:([a-zA-Z0-9.-]+)$/;

export const imageNameRegex = /^([\w.-]+)\/([\w.-]+):([\w.-]+)$/;
export const imageTagRegex = /:([\w.-]+)$/;

export function parseImageLink(imageLink: string) {
  const parts = imageLink.split(":");
  if (parts.length === 2) {
    const nameParts = parts[0].split("/");
    const imageName =
      nameParts.length > 1 ? nameParts[nameParts.length - 1] : nameParts[0];
    const imageTag = parts[1];
    return { imageName, imageTag };
  }
  return { imageName: "", imageTag: "" };
}

export const apiUserImagepackCreate = async (imagepack: ImagePackCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/images/create",
    imagepack,
  );
  return response.data;
};

export const apiUserImagepackUpload = async (imageupload: ImagePackUpload) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/images/upload",
    imageupload,
  );
  return response.data;
};

export enum ImagePackListType {
  Create = 1,
  Upload,
}

export const apiUserImagePackList = (type: number) =>
  instance.get<IResponse<ImagePackListResponse>>(
    `${VERSION}/images/list?type=${type}`,
  );

export const apiUserImageCreateGet = (id: string) =>
  instance.get<IResponse<ImagePackInfoResponse>>(
    `${VERSION}/images/get?id=${id}`,
  );

export const apiUserImageCreateLog = (id: string) =>
  instance.get<IResponse<ImagePackLogResponse>>(
    `${VERSION}/images/log?id=${id}`,
  );

export interface ImageDeleteRequest {
  id: number;
  imagetype: number;
}

export const apiUserImagePackDelete = async (req: ImageDeleteRequest) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/images/delete",
    req,
  );
  return response.data;
};
