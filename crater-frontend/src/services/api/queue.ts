import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { AccessMode, Role } from "./auth";

export interface QueueBasic {
  name: string;
  nickname: string;
  role: Role;
  access: AccessMode;
  expiredAt?: Date;
}

export const apiQueueList = () =>
  instance.get<IResponse<QueueBasic[]>>(`${VERSION}/accounts`);
