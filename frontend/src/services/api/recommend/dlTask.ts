import instance, { VERSION } from "../../axios";
import { IResponse } from "../../types";

export interface IDlAnalyze {
  p100: {
    gpuUtilAvg: number;
    gpuMemoryMaxGB: number;
  };
  v100: {
    gpuUtilAvg: number;
    gpuMemoryMaxGB: number;
    smActiveAvg: number;
    smOccupancyAvg: number;
    fp32ActiveAvg: number;
    dramActiveAvg: number;
  };
}

interface IDlAnalyzeRequest {
  runningType: string;
  relationShips: string[];
  macs: number;
  params: number;
  batchSize: number;
  vocabularySize: number[];
  embeddingDim: number[];
}

// /v1/recommenddljob/analyze
export const apiDlAnalyze = (data: IDlAnalyzeRequest) =>
  instance.post<IResponse<IDlAnalyze>>(VERSION + "/recommenddljob/analyze", {
    ...data,
  });
