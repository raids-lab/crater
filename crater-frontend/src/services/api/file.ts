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
  instance.get<IResponse<FileItem[] | undefined>>(`ss/files/${path}`);

export const apiGetUserFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(`ss/admin/user/${path}`);

export const apiGetQueueFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(`ss/admin/queue/${path}`);

export const apiMkdir = async (path: string) => {
  await instance.request({
    method: "MKCOL",
    url: `ss/${path}`,
  });
};

export const apiUploadFile = (path: string, filedata: ArrayBuffer) =>
  instance.put(`ss/${path}`, filedata);
