import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { IClusterNodeInfo } from "./cluster";

export enum NodeType {
  Hygon = "hygon",
  Shenwei = "shenwei",
  Yitian = "yitian",
}

export const apiGetNodes = () =>
  instance.get<
    IResponse<{
      rows: IClusterNodeInfo[];
    }>
  >(VERSION + "/nodes");
