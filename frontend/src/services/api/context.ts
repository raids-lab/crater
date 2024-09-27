import { globalSettings, store } from "@/utils/store";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

interface ResourceBase {
  amount: number;
  format: string;
}

export interface ResourceResp {
  label: string;
  allocated?: ResourceBase;
  guarantee?: ResourceBase;
  deserved?: ResourceBase;
  capability?: ResourceBase;
}

export interface QuotaResp {
  cpu: ResourceResp;
  memory: ResourceResp;
  gpus: ResourceResp[];
}

const { scheduler } = store.get(globalSettings);

export const apiContextQuota = () => {
  const url = scheduler === "volcano" ? "/context/quota" : "/aijobs/quota";
  return instance.get<IResponse<QuotaResp>>(VERSION + url);
};
