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
  type?: "gpu" | "rdma";
  networks?: Resource[];
}

export const apiResourceList = (withVendorDomain: boolean) => {
  return instance.get<IResponse<Resource[]>>(`${VERSION}/resources`, {
    params: {
      withVendorDomain,
    },
  });
};

// @Router /v1/resources/{id}/networks [get]
export const apiResourceNetworks = (id: number) => {
  return instance.get<IResponse<Resource[]>>(
    `${VERSION}/resources/${id}/networks`,
  );
};

// @Router /v1/admin/resources/sync [post]
export const apiAdminResourceSync = () => {
  return instance.post<IResponse<never>>(`${VERSION}/admin/resources/sync`);
};

// @Router /v1/admin/resources/{id} [put]
export const apiAdminResourceUpdate = (
  id: number,
  label?: string,
  type?: "gpu" | "rdma" | "default" | null,
) => {
  return instance.put<IResponse<Resource>>(`${VERSION}/admin/resources/${id}`, {
    label,
    type,
  });
};

// @Router /v1/admin/resources/{id} [delete]
export const apiAdminResourceDelete = (id: number) => {
  return instance.delete<IResponse<Resource>>(
    `${VERSION}/admin/resources/${id}`,
  );
};

// @Router /v1/admin/resources/{id}/networks [get]
export const apiAdminResourceNetworksList = (id: number) => {
  return instance.get<IResponse<Resource[]>>(
    `${VERSION}/admin/resources/${id}/networks`,
  );
};

// @Router /v1/admin/resources/{id}/networks [post]
export const apiAdminResourceNetworkAdd = (id: number, rdmaId: number) => {
  return instance.post<IResponse<Resource>>(
    `${VERSION}/admin/resources/${id}/networks`,
    {
      rdmaId,
    },
  );
};

// @Router /v1/admin/resources/{id}/networks/{networkId} [delete]
export const apiAdminResourceNetworkDelete = (
  id: number,
  networkId: number,
) => {
  return instance.delete<IResponse<Resource>>(
    `${VERSION}/admin/resources/${id}/networks/${networkId}`,
  );
};
