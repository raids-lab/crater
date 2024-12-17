import instance from "../axios";
import { IResponse } from "@/services/types";

export interface FileItem {
  isdir: boolean;
  modifytime: string;
  name: string;
  size: number;
  sys?: never;
}

export interface UserSpace {
  username: string;
  space: string;
}
export interface QeueuSpace {
  queuename: string;
  space: string;
}
export const apiGetFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(`ss/files/${path}`);

export const apiGetUserFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(`ss/admin${path}`);

export const apiGetQueueFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(`ss/admin${path}`);

export const apiMkdir = async (path: string) => {
  await instance.request({
    method: "MKCOL",
    url: `ss/${path}`,
  });
};

export const apiUploadFile = (path: string, filedata: ArrayBuffer) =>
  instance.put(`ss/${path}`, filedata);

export const apiFileDelete = (path: string) =>
  instance.delete<IResponse<string>>(`/ss/delete/${path}`);

export const apiGetUserSpace = () =>
  instance.get<IResponse<UserSpace[] | undefined>>(`ss/userspace`);

export const apiGetQueueSpace = () =>
  instance.get<IResponse<QeueuSpace[] | undefined>>(`ss/queuespace`);
