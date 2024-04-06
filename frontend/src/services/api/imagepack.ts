import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import {
  CheckCircledIcon,
  CircleIcon,
  ClockIcon,
  CrossCircledIcon,
  StopwatchIcon,
} from "@radix-ui/react-icons";

export type ImagePackInfoResponse = {
  ID: number;
  // name: string;
  imagelink: string;
  status: string;
  createdAt: string;
  nametag: string;
  username: string;
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

export type ImagePackInfo = {
  id: number;
  nametag: string;
  link: string;
  username: string;
  status: string;
  createdAt: string;
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

export interface ImagePackCreate {
  gitRepository: string;
  accessToken: string;
  registryServer: string;
  registryUser: string;
  registryPass: string;
  registryProject: string;
  imageName: string;
  imageTag: string;
  needProfile: boolean;
}

export const apiUserImagepackCreate = async (imagepack: ImagePackCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/image/create",
    imagepack,
  );
  return response.data;
};

export const apiUserImagePackList = () =>
  instance.get<IResponse<ImagePackInfoResponse[]>>(VERSION + "/image/list");

export const apiUserImagePackDelete = async (id: number) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/image/deleteid",
    {
      id,
    },
  );
  return response.data;
};
