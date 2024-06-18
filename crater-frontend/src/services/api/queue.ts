import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { AccessMode, Role } from "./auth";

export interface QueueBasic {
  id: string;
  name: string;
  role: Role;
  access: AccessMode;
}

export const apiQueueList = () =>
  instance.get<IResponse<QueueBasic[]>>(`${VERSION}/context/queues`);
