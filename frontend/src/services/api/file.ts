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

export interface MoveFile {
  fileName: string;
  dst: string;
}

export const apiGetFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(
    `ss/files/${path.replace(/^\//, "")}`,
  );

export const apiGetAdminFile = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(
    `ss/admin/files/${path.replace(/^\//, "")}`,
  );

export const apiGetUserFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(
    `ss/admin/${path.replace(/^\//, "")}`,
  );

export const apiGetQueueFiles = (path: string) =>
  instance.get<IResponse<FileItem[] | undefined>>(
    `ss/admin/${path.replace(/^\//, "")}`,
  );

export const apiMkdir = async (path: string) => {
  await instance.request({
    method: "MKCOL",
    url: `ss/${path.replace(/^\//, "")}`,
  });
};

export const apiFileDelete = (path: string) =>
  instance.delete<IResponse<string>>(`/ss/delete/${path.replace(/^\//, "")}`);

export const apiGetUserSpace = () =>
  instance.get<IResponse<UserSpace[] | undefined>>(`ss/userspace`);

export const apiGetQueueSpace = () =>
  instance.get<IResponse<QeueuSpace[] | undefined>>(`ss/queuespace`);

export const apiMoveFile = (req: MoveFile, path: string) =>
  instance.post<IResponse<MoveFile>>(`ss/move/${path.replace(/^\//, "")}`, req);
