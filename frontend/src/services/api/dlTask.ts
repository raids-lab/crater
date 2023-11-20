import { KubernetesResource } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

interface IDlTask {
  name: string;
  namespace: string;
  creationTimestamp: string;
  spec: {
    replicas: number;
    runningType: string;
    datasets: {
      name: string;
    }[];
    releationShips: unknown[];
    template: {
      metadata: {
        creationTimestamp: string;
      };
      spec: {
        containers: {
          name: string;
          image: string;
          resources: {
            limits: KubernetesResource;
          };
        }[];
      };
    };
    username: string;
    macs: number;
    params: number;
    batchSize: number;
    embeddingSizeTotal: number;
    embeddingDimTotal: number;
    embeddingTableCount: number;
    vocabularySize: number[];
    embeddingDim: number[];
    inputTensor: string;
  };
  status: {
    phase: string;
    podNames: string[];
  };
}

// /v1/recommenddljob/list
export const apiDlTaskList = () =>
  instance.get<IResponse<IDlTask[]>>(VERSION + "/recommenddljob/list");

// {
//   "name": "test-recommenddljob", // 任务名称，必填
//   "replicas": 1, // pod副本个数，必填
//   "runningType": "one-shot", // 运行模式，两种取值：one-shot/long-running，必填
//   "datasets": [], // 数据集，可为空，可以选datasets/list接口里返回的数据集，列表内每个元素仅有一个name字段
//   "relationShips": [], // 暂时不管
//   "macs": 230006514, // 模型乘加运算数Macs，必填
//   "params": 66665211, // 模型参数量，必填
//   "batch_size": 64, // BatchSize， 必填
//   "vocabularySize": [5000000,10000000,600000], // 稀疏特征输入维度，前端建议和embeddingDim设计为kv形式，即列表的长度必须和embeddingDim长度一致，且元素一一对应
//   "embeddingDim": [32,32,32], // 稀疏特征输出维度
//   "template": { // pod 模板
//       "spec": {
//           "containers": [
//               {
//                   "name": "test", // 容器名称
//                   "image": "nginx:latest", // 镜像地址
//                   "resources": {
//                       "limits": {
//                           "nvidia.com/gpu": 3 // gpu个数
//                       }
//                   }
//               }
//           ]
//       }
//   }
// }
export interface IDlTaskCreate {
  name: string;
  replicas: number;
  runningType: string;
  datasets: {
    name: string;
  }[];
  relationShips: unknown[];
  macs: number;
  params: number;
  batchSize: number;
  vocabularySize: number[];
  embeddingDim: number[];
  template: {
    spec: {
      containers: {
        name: string;
        image: string;
        resources: {
          limits: KubernetesResource;
        };
      }[];
    };
  };
}

// /v1/recommenddljob/create
export const apiDlTaskCreate = (data: IDlTaskCreate) => {
  return instance.post<IResponse<string>>(
    VERSION + "/recommenddljob/create",
    data,
  );
};

// /v1/recommenddljob/info?name=test-recommenddljob
export const apiDlTaskInfo = (name: string) =>
  instance.get<IResponse<IDlTask>>(VERSION + "/recommenddljob/info", {
    params: {
      name,
    },
  });

// v1/recommenddljob/delete?name=test-recommenddljob
export const apiDlTaskDelete = (name: string) =>
  instance.post<IResponse<string>>(VERSION + "/recommenddljob/delete", {
    name,
  });
