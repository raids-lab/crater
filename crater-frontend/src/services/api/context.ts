import { KubernetesResource } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

export interface Quota {
  capability: KubernetesResource;
  allocated: KubernetesResource;
}

export const DefaultQuota: Quota = {
  capability: {
    cpu: 0,
    memory: "0",
    "nvidia.com/gpu": 0,
  },
  allocated: {
    cpu: 0,
    memory: "0",
    "nvidia.com/gpu": 0,
  },
};

export const apiContextQuota = () =>
  instance.get<IResponse<Quota>>(VERSION + "/context/queue");
