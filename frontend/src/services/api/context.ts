import { globalSettings, store } from "@/utils/store";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { IUserAttributes } from "./admin/user";

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

export const apiContextUpdateUserAttributes = (data: IUserAttributes) =>
  instance.put<IResponse<string>>(`${VERSION}/context/attributes`, data);

// apiSendVerificationEmail,
// apiVerifyEmailCode,

export const apiSendVerificationEmail = (email: string) =>
  instance.post<IResponse<string>>(`${VERSION}/context/email/code`, { email });

export const apiVerifyEmailCode = (email: string, code: string) =>
  instance.post<IResponse<string>>(`${VERSION}/context/email/update`, {
    code,
    email,
  });
