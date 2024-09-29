import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

export interface Resource {
  ID: number;
  name: string;
  vendorDomain: string;
  resourceType: string;
  amount: number;
  amountSingleMax: number;
  format: string;
  priority: number;
  label: string;
}

export const apiResourceList = (withVendorDomain: boolean) => {
  return instance.get<IResponse<Resource[]>>(`${VERSION}/resources`, {
    params: {
      withVendorDomain,
    },
  });
};

// @Router /v1/admin/resources/sync [post]
export const apiAdminResourceSync = () => {
  return instance.post<IResponse<never>>(`${VERSION}/admin/resources/sync`);
};

// @Router /v1/admin/resources/{id} [put]
export const apiAdminResourceUpdate = (id: number, label: string) => {
  return instance.put<IResponse<Resource>>(`${VERSION}/admin/resources/${id}`, {
    label,
  });
};
