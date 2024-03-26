import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";

type ImagePackInfoResponse = {
  ID: number;
  // name: string;
  imagelink: string;
  status: string;
  createdAt: string;
  nametag: string;
};

export const apiImagePackList = () =>
  instance.get<IResponse<ImagePackInfoResponse[]>>(VERSION + "/image/list");

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

export const apiImagepackCreate = async (task: ImagePackCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/image/create",
    task,
  );
  return response.data;
};
