import { globalJobUrl, store } from "@/utils/store";
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

const JOB_URL = store.get(globalJobUrl);

export const apiContextQuota = () => {
  const url = JOB_URL === "aijobs" ? "/aijobs/quota" : "/context/quota";
  return instance.get<IResponse<QuotaResp>>(VERSION + url);
};
