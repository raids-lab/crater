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
};

export type ImagePackInfo = {
  id: number;
  nametag: string;
  link: string;
  status: string;
  createdAt: string;
};

type ImagePackStatusValue =
  | "Initial"
  | "Pending"
  | "Running"
  | "Finished"
  | "Failed";

export const imagepack_statuses: {
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

interface ImagePackCreate {
  gitRepository: string;
  accessToken: string;
  registryServer: string;
  registryUser: string;
  registryPass: string;
  registryProject: string;
  imageName: string;
  imageTag: string;
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

export const apiAdminImagepackCreate = async (imagepack: ImagePackCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/admin/images/create",
    imagepack,
  );
  return response.data;
};

export const apiAdminPublicImagePackList = () =>
  instance.get<IResponse<ImagePackInfoResponse[]>>(
    VERSION + "/admin/images/public",
  );

export const apiAdminPersonalImagePackList = () =>
  instance.get<IResponse<ImagePackInfoResponse[]>>(
    VERSION + "/admin/images/personal",
  );
