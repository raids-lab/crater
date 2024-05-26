import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

interface ResourceBase {
  amount: number;
  format: string;
}

export interface ResourceResp {
  label: string;
  allocated: ResourceBase;
  guarantee?: ResourceBase;
  deserved?: ResourceBase;
  capability: ResourceBase;
}

export interface QuotaResp {
  cpu: ResourceResp;
  memory: ResourceResp;
  gpus: ResourceResp[];
}

export const apiContextQuota = () =>
  instance.get<IResponse<QuotaResp>>(VERSION + "/context/queue");
