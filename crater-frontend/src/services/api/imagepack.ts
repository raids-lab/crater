import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import {
  CheckCircledIcon,
  CircleIcon,
  ClockIcon,
  CrossCircledIcon,
  StopwatchIcon,
} from "@radix-ui/react-icons";
import { IUserInfo, JobType } from "./vcjob";

export type ListKanikoResponse = {
  kanikoList: KanikoInfoResponse[];
};

export type KanikoInfoResponse = {
  ID: number;
  imageLink: string;
  status: ImagePackStatus;
  buildSource: ImagePackSource;
  createdAt: string;
  size: number;
  dockerfile: string;
  description: string;
  podName: string;
  podNameSpace: string;
  userInfo: IUserInfo;
  tags: string[];
  imagepackName: string;
};

export type ListImageResponse = {
  imageList: ImageInfoResponse[];
};

export type ImageInfoResponse = {
  ID: number;
  imageLink: string;
  description: string;
  status: string;
  createdAt: string;
  isPublic: boolean;
  taskType: JobType;
  userInfo: IUserInfo;
  tags: string[];
  imageBuildSource: imagepackSourceTypeValue;
  imagepackName: string;
};

export type ProjectCredentialResponse = {
  name: string;
  password: string;
  exist: boolean;
};

export type ProjectDetailResponse = {
  quota: number;
  used: number;
  total: number;
  project: string;
};

export const getHeader = (key: string): string => {
  switch (key) {
    case "image":
      return "镜像";
    case "userInfo":
      return "提交者";
    case "status":
      return "状态";
    case "createdAt":
      return "创建时间";
    case "taskType":
      return "类型";
    case "imageType":
      return "镜像类型";
    case "isPublic":
      return "可见性";
    case "size":
      return "大小";
    default:
      return key;
  }
};

export type ImagePackStatus =
  | "Initial"
  | "Pending"
  | "Running"
  | "Finished"
  | "Failed"
  | "";

export enum ImagePackSource {
  Dockerfile = "Dockerfile",
  PipApt = "PipApt",
  Snapshot = "Snapshot",
  EnvdAdvanced = "EnvdAdvanced",
  EnvdRaw = "EnvdRaw",
}

export const imagepackStatuses: {
  value: ImagePackStatus;
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
  description: string;
  image: string;
  packages: string;
  requirements: string;
  name: string;
  tag: string;
  tags: string[];
  template: string;
  buildSource: ImagePackSource;
}

export interface DockerfileCreate {
  description: string;
  dockerfile: string;
  name: string;
  tag: string;
  tags: string[];
  template: string;
  buildSource: ImagePackSource;
}

export interface EnvdCreate {
  description: string;
  envd: string;
  name: string;
  tag: string;
  python: string;
  base: string;
  tags: string[];
  template: string;
  buildSource: ImagePackSource;
}

export interface ImageUpload {
  imageLink: string;
  imageName: string;
  imageTag: string;
  description: string;
  taskType: JobType;
  tags: string[];
}

export interface UpdateDescription {
  id: number;
  description: string;
}

export interface UpdateImageTag {
  id: number;
  tags: string[];
}

export interface UpdateTaskType {
  id: number;
  taskType: JobType;
}

export interface ImageLinkPair {
  id: number;
  imageLink: string;
  description: string;
  creator: IUserInfo;
}

export interface ImageLinkPairs {
  linkPairs: ImageLinkPair[];
}

export interface ImageIDList {
  idList: number[];
}

export interface HarborIPData {
  ip: string;
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

export const ImageDefaultTags = [
  { value: "CUDA" },
  { value: "Tensorflow" },
  { value: "Pytorch" },
  { value: "Jupyter" },
  { value: "Ray" },
  { value: "Python" },
  { value: "NO-CUDA" },
  { value: "vLLM" },
];

export const imageLinkRegex =
  /^[a-zA-Z0-9.-]+(\/[a-zA-Z0-9_.-]+)+:[a-zA-Z0-9_](?:[a-zA-Z0-9_.-]*[a-zA-Z0-9_])?$/;

export const imageNameRegex = /^[a-z0-9]+(?:[._-][a-z0-9]+)*$/;
export const imageTagRegex =
  /^(?=.{1,128}$)(?![-.])([a-zA-Z0-9_.+-]*)(?<![-.])$/;

export const dockerfileImageLinkRegex =
  /^FROM\s+(?:--platform=[^\s]+\s+)?([^\s#]+)/gim;

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

export const apiUserCreateByDockerfile = async (
  imageDockerfile: DockerfileCreate,
) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/images/dockerfile",
    imageDockerfile,
  );
  return response.data;
};

export const apiUserCreateByEnvd = async (envdInfo: EnvdCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/images/envd",
    envdInfo,
  );
  return response.data;
};

export const apiUserDeleteKaniko = (id: number) =>
  instance.delete<IResponse<string>>(VERSION + `/images/kaniko/${id}`);

export const apiUserDeleteKanikoList = (idList: number[]) =>
  instance.post<IResponse<string>>(`${VERSION}/images/deletekaniko`, {
    idList,
  });

export const apiUserGetKaniko = (name: string) =>
  instance.get<IResponse<KanikoInfoResponse>>(
    `${VERSION}/images/getbyname?name=${name}`,
  );

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

export const apiUserDeleteImageList = (idList: number[]) =>
  instance.post<IResponse<string>>(`${VERSION}/images/deleteimage`, { idList });

export const apiUserGetCredential = () =>
  instance.post<IResponse<ProjectCredentialResponse>>(
    `${VERSION}/images/credential`,
  );

export const apiUserGetQuota = () =>
  instance.get<IResponse<ProjectDetailResponse>>(`${VERSION}/images/quota`);

export const apiUserChangeImageDescription = (data: UpdateDescription) =>
  instance.post<IResponse<string>>(`${VERSION}/images/description`, data);

export const apiUserChangeImageTaskType = (data: UpdateTaskType) =>
  instance.post<IResponse<string>>(`${VERSION}/images/type`, data);

export const apiUserCheckImageValid = (linkPairs: ImageLinkPairs) =>
  instance.post<IResponse<ImageLinkPairs>>(
    `${VERSION}/images/valid`,
    linkPairs,
  );

export const apiGetHarborIP = () =>
  instance.get<IResponse<HarborIPData>>(`${VERSION}/images/harbor`);

export const apiUserUpdateImageTags = (data: UpdateImageTag) =>
  instance.post<IResponse<string>>(`${VERSION}/images/tags`, data);

export const apiUserGetImageTemplate = (name: string) =>
  instance.get<IResponse<string>>(`${VERSION}/images/template?name=${name}`);
