import instance from "../axios";
import { IResponse } from "@/services/types";

export interface FileItem {
  isdir: boolean;
  modifytime: string;
  name: string;
  size: number;
  sys?: never;
}

export const apiGetFiles = (path: string) =>
  instance.get<IResponse<FileItem[]>>(`ss/files/${path}`);

export const apiMkdir = async (path: string) => {
  await instance.request({
    method: "MKCOL",
    url: `ss/${path}`,
  });
};

export const apiUploadFile = (path: string, filedata: ArrayBuffer) =>
  instance.put(`ss/${path}`, filedata);
