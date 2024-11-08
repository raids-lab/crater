import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import {
  CheckCircledIcon,
  CircleIcon,
  ClockIcon,
  CrossCircledIcon,
  StopwatchIcon,
} from "@radix-ui/react-icons";
import { JobType } from "./vcjob";

export type ListKanikoResponse = {
  kanikolist: KanikoInfoResponse[];
  totalsize: number;
};
type KanikoInfoResponse = {
  ID: number;
  // name: string;
  imagelink: string;
  status: string;
  createdAt: string;
  ispublic: boolean;
};
export type KanikoInfo = {
  id: number;
  link: string;
  status: string;
  createdAt: string;
};

export type ListImageResponse = {
  imagelist: ImageInfoResponse[];
};
type ImageInfoResponse = {
  ID: number;
  // name: string;
  imagelink: string;
  status: string;
  createdAt: string;
  ispublic: boolean;
  tasktype: JobType;
  creatorname: string;
};
export type ImageInfo = {
  id: number;
  link: string;
  status: string;
  createdAt: string;
  ispublic: boolean;
  tasktype: JobType;
  creatorname: string;
};

export type KanikoLogResponse = {
  content: string;
};

export const getHeader = (key: string): string => {
  switch (key) {
    case "nametag":
      return "名称";
    case "link":
      return "镜像地址";
    case "creatorname":
      return "提交者";
    case "status":
      return "状态";
    case "createdAt":
      return "创建时间";
    case "tasktype":
      return "任务类型";
    case "imagetype":
      return "镜像类型";
    case "ispublic":
      return "公私类型";
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
    label: "kuberay",
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
    label: "custom",
  },
];

export type imagepackSourceTypeValue = 1 | 2;

export const imagepackSourceType: {
  value: imagepackSourceTypeValue;
  label: string;
}[] = [
  {
    value: 1,
    label: "镜像制作",
  },
  {
    value: 2,
    label: "镜像上传",
  },
];

export type imagepackPublicPersonalStatusValue = false | true;
export const imagepackPublicPersonalStatus: {
  value: imagepackPublicPersonalStatusValue;
  label: string;
}[] = [
  {
    value: false,
    label: "私有",
  },
  {
    value: true,
    label: "公共",
  },
];

export interface KanikoCreate {
  gitRepository: string;
  accessToken: string;
  dockerfile: string;
  registryServer: string;
  registryUser: string;
  registryPass: string;
  registryProject: string;
  imageName: string;
  imageTag: string;
  description: string;
}

export interface ImageUpload {
  imageLink: string;
  imageName: string;
  imageTag: string;
  description: string;
  taskType: JobType;
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

export const apiUserListKaniko = () =>
  instance.get<IResponse<ListKanikoResponse>>(`${VERSION}/images/kaniko`);

export const apiUserCreateKaniko = async (imagepack: KanikoCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/images/kaniko",
    imagepack,
  );
  return response.data;
};

export const apiUserDeleteKaniko = (id: number) =>
  instance.delete<IResponse<string>>(VERSION + `/images/kaniko/${id}`);

export const apiUserGetKaniko = (id: string) =>
  instance.get<IResponse<KanikoInfoResponse>>(`${VERSION}/images/get?id=${id}`);

export const apiUserLogKaniko = (id: string) =>
  instance.get<IResponse<KanikoLogResponse>>(`${VERSION}/images/log?id=${id}`);

export const apiUserListImage = () =>
  instance.get<IResponse<ListImageResponse>>(`${VERSION}/images/image`);

export const apiUserChangeImagePublicStatus = (id: number) =>
  instance.post<IResponse<string>>(VERSION + `/images/change/${id}`);

export const apiUserUploadImage = async (imageupload: ImageUpload) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/images/image",
    imageupload,
  );
  return response.data;
};

export const apiUserDeleteImage = (id: number) =>
  instance.delete<IResponse<string>>(VERSION + `/images/image/${id}`);
