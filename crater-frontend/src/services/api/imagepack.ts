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
  kanikoList: KanikoInfoResponse[];
  totalSize: number;
};
type KanikoInfoResponse = {
  ID: number;
  imageLink: string;
  status: string;
  createdAt: string;
  dockerfile: string;
  description: string;
  podName: string;
  podNameSpace: string;
};
export type KanikoInfo = {
  id: number;
  imageLink: string;
  status: string;
  createdAt: string;
  podName: string;
  podNameSpace: string;
};

export type ListImageResponse = {
  imageList: ImageInfoResponse[];
};
export type ImageInfoResponse = {
  ID: number;
  // name: string;
  imageLink: string;
  description: string;
  status: string;
  createdAt: string;
  isPublic: boolean;
  taskType: JobType;
  creatorName: string;
};
export type ImageInfo = {
  id: number;
  imageLink: string;
  status: string;
  createdAt: string;
  isPublic: boolean;
  taskType: JobType;
  creatorName: string;
};

export type KanikoLogResponse = {
  content: string;
};

export type ProjectCredentialResponse = {
  name: string;
  password: string;
  exist: boolean;
};

export const getHeader = (key: string): string => {
  switch (key) {
    case "imageLink":
      return "镜像地址";
    case "creatorName":
      return "提交者";
    case "status":
      return "状态";
    case "createdAt":
      return "创建时间";
    case "taskType":
      return "任务类型";
    case "imageType":
      return "镜像类型";
    case "isPublic":
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
  instance.get<IResponse<KanikoInfoResponse>>(
    `${VERSION}/images/getbyid?id=${id}`,
  );

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

export const apiUserGetCredential = () =>
  instance.post<IResponse<ProjectCredentialResponse>>(
    `${VERSION}/images/credential`,
  );
