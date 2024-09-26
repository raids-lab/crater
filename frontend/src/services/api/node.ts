import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { IClusterNodeInfo } from "./cluster";

export const apiGetNodes = () =>
  instance.get<
    IResponse<{
      rows: IClusterNodeInfo[];
    }>
  >(VERSION + "/nodes");
