import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

interface Quota {
  jobReq: number;
  job: number;
  nodeReq: number;
  node: number;
  cpuReq: number;
  cpu: number;
  gpuReq: number;
  gpu: number;
  memReq: number;
  mem: number;
  gpuMemReq: number;
  gpuMem: number;
  storage: number;
  extra: string | null;
}

export const apiContextQuota = () =>
  instance.get<IResponse<Quota>>(VERSION + "/context/quota");
