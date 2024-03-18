import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { KubernetesResource } from "@/utils/resource";

export interface IClusterNodeInfo {
  name: string;
  role: string;
  labels: Record<string, string>;
  isReady: boolean;
  capacity: KubernetesResource;
  allocated: KubernetesResource;
}

export const apiGetAdminNodes = () =>
  instance.get<
    IResponse<{
      rows: IClusterNodeInfo[];
    }>
  >(VERSION + "/admin/nodes");
