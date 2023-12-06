import { KubernetesResource } from "@/utils/resource";
import instance, { VERSION } from "../../axios";
import { IResponse } from "../../types";

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
  datasets: string[];
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

// Pod define
// {
//   "kind": "Pod",
//   "apiVersion": "v1",
//   "metadata": {
//       "name": "test-recommenddljob-25f3a429-0927-4b0c-a245-955dcb42c8a0-0",
//       "namespace": "user-lpxx",
//       "uid": "2e6f4a83-100a-4c49-a00a-fe5387de27ad",
//       "resourceVersion": "26061774",
//       "creationTimestamp": "2023-11-18T12:35:39Z",
//       "labels": {
//           "kube-gpu-sparse/job_name": "test-recommenddljob"
//       },
//       "annotations": {
//           "kube-gpu-sparse/releation_ship": "null",
//           "kube-gpu-sparse/resource_analyze": "{\"end2end\":{\"P100\":{\"gpu_util_avg\":88.98382,\"gpu_util_std\":4.1917596,\"mem_usage\":3.1259863},\"2080\":{\"gpu_util_avg\":85.38019,\"gpu_util_std\":3.9001899,\"mem_usage\":6.498702},\"TITAN\":{\"gpu_util_avg\":90.51219,\"gpu_util_std\":2.6988258,\"mem_usage\":4.4265566},\"V100\":{\"gpu_util_avg\":89.465355,\"gpu_util_std\":3.8834035,\"mem_usage\":17.204525,\"sm_active_avg\":84.95571,\"sm_active_std\":3.448447,\"sm_occupied_avg\":73.74939,\"sm_occupied_std\":3.6292644,\"dram_active_avg\":77.536896,\"dram_active_std\":2.9764752,\"fp32_active_avg\":4.9109125,\"fp32_active_std\":0.4651995}},\"not_end2end\":null}",
//           "kube-gpu-sparse/task_info": "{\"task_id\":0,\"job_uuid\":\"25f3a429-0927-4b0c-a245-955dcb42c8a0\",\"is_end2end\":true,\"macs\":230.00652,\"params\":66665.21,\"batch_size\":0,\"embedding_size_total\":1560,\"embedding_dim_total\":96,\"embedding_table_count\":3,\"related_pods\":[],\"datasets\":[]}"
//       }
//   },
//   "spec": {
//       "volumes": [
//           {
//               "name": "kube-api-access-g277w",
//               "projected": {
//                   "sources": [
//                       {
//                           "serviceAccountToken": {
//                               "expirationSeconds": 3607,
//                               "path": "token"
//                           }
//                       },
//                       {
//                           "configMap": {
//                               "name": "kube-root-ca.crt",
//                               "items": [
//                                   {
//                                       "key": "ca.crt",
//                                       "path": "ca.crt"
//                                   }
//                               ]
//                           }
//                       },
//                       {
//                           "downwardAPI": {
//                               "items": [
//                                   {
//                                       "path": "namespace",
//                                       "fieldRef": {
//                                           "apiVersion": "v1",
//                                           "fieldPath": "metadata.namespace"
//                                       }
//                                   }
//                               ]
//                           }
//                       }
//                   ],
//                   "defaultMode": 420
//               }
//           }
//       ],
//       "containers": [
//           {
//               "name": "test",
//               "image": "nginx:latest",
//               "env": [
//                   {
//                       "name": "NVIDIA_VISIBLE_DEVICES",
//                       "valueFrom": {
//                           "configMapKeyRef": {
//                               "name": "test-recommenddljob-25f3a429-0927-4b0c-a245-955dcb42c8a0-0-nvidia-gpu-env",
//                               "key": "NVIDIA_VISIBLE_DEVICES"
//                           }
//                       }
//                   }
//               ],
//               "resources": {
//                   "limits": {
//                       "nvidia.com/gpu": "3"
//                   },
//                   "requests": {
//                       "nvidia.com/gpu": "3"
//                   }
//               },
//               "volumeMounts": [
//                   {
//                       "name": "kube-api-access-g277w",
//                       "readOnly": true,
//                       "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
//                   }
//               ],
//               "terminationMessagePath": "/dev/termination-log",
//               "terminationMessagePolicy": "File",
//               "imagePullPolicy": "Always"
//           }
//       ],
//       "restartPolicy": "Never",
//       "terminationGracePeriodSeconds": 30,
//       "dnsPolicy": "ClusterFirst",
//       "serviceAccountName": "default",
//       "serviceAccount": "default",
//       "nodeName": "dell-66",
//       "securityContext": {},
//       "schedulerName": "kube-gpu-sparse-scheduler",
//       "tolerations": [
//           {
//               "key": "node.kubernetes.io/not-ready",
//               "operator": "Exists",
//               "effect": "NoExecute",
//               "tolerationSeconds": 300
//           },
//           {
//               "key": "node.kubernetes.io/unreachable",
//               "operator": "Exists",
//               "effect": "NoExecute",
//               "tolerationSeconds": 300
//           }
//       ],
//       "priority": 0,
//       "enableServiceLinks": true,
//       "preemptionPolicy": "PreemptLowerPriority"
//   },
//   "status": {
//       "phase": "Running",
//       "conditions": [
//           {
//               "type": "Initialized",
//               "status": "True",
//               "lastProbeTime": null,
//               "lastTransitionTime": "2023-11-18T12:35:40Z"
//           },
//           {
//               "type": "Ready",
//               "status": "True",
//               "lastProbeTime": null,
//               "lastTransitionTime": "2023-11-18T12:35:45Z"
//           },
//           {
//               "type": "ContainersReady",
//               "status": "True",
//               "lastProbeTime": null,
//               "lastTransitionTime": "2023-11-18T12:35:45Z"
//           },
//           {
//               "type": "PodScheduled",
//               "status": "True",
//               "lastProbeTime": null,
//               "lastTransitionTime": "2023-11-18T12:35:40Z"
//           }
//       ],
//       "hostIP": "192.168.5.66",
//       "podIP": "10.244.9.161",
//       "podIPs": [
//           {
//               "ip": "10.244.9.161"
//           }
//       ],
//       "startTime": "2023-11-18T12:35:40Z",
//       "containerStatuses": [
//           {
//               "name": "test",
//               "state": {
//                   "running": {
//                       "startedAt": "2023-11-18T12:35:44Z"
//                   }
//               },
//               "lastState": {},
//               "ready": true,
//               "restartCount": 0,
//               "image": "nginx:latest",
//               "imageID": "docker-pullable://nginx@sha256:86e53c4c16a6a276b204b0fd3a8143d86547c967dc8258b3d47c3a21bb68d3c6",
//               "containerID": "docker://dc6016da5c673f9f2423737f1baa468507207da455c84129f05a956872b45e7d",
//               "started": true
//           }
//       ],
//       "qosClass": "BestEffort"
//   }
// }
interface IDlTaskPod {
  kind: string;
  apiVersion: string;
  metadata: {
    name: string;
    namespace: string;
    uid: string;
    resourceVersion: string;
    creationTimestamp: string;
    labels: {
      "kube-gpu-sparse/job_name": string;
    };
    annotations: {
      "kube-gpu-sparse/releation_ship": string;
      "kube-gpu-sparse/resource_analyze": string;
      "kube-gpu-sparse/task_info": string;
    };
  };
  spec: {
    volumes: unknown[];
    containers: {
      name: string;
      image: string;
      env: unknown[];
      resources: {
        limits: KubernetesResource;
        requests: KubernetesResource;
      };
      volumeMounts: unknown[];
      terminationMessagePath: string;
      terminationMessagePolicy: string;
      imagePullPolicy: string;
    }[];
    restartPolicy: string;
    terminationGracePeriodSeconds: number;
    dnsPolicy: string;
    serviceAccountName: string;
    serviceAccount: string;
    nodeName: string;
    securityContext: unknown;
    schedulerName: string;
    tolerations: unknown[];
    priority: number;
    enableServiceLinks: boolean;
    preemptionPolicy: string;
  };
  status: {
    phase: string;
    conditions: {
      type: string;
      status: string;
      lastProbeTime: unknown;
      lastTransitionTime: string;
    }[];
    hostIP: string;
    podIP: string;
    podIPs: {
      ip: string;
    }[];
    startTime: string;
    containerStatuses: {
      name: string;
      state: {
        running: {
          startedAt: string;
        };
      };
      lastState: unknown;
      ready: boolean;
      restartCount: number;
      image: string;
      imageID: string;
      containerID: string;
      started: boolean;
    }[];
    qosClass: string;
  };
}

// /v1/recommenddljob/pods?name=test-recommenddljob
export const apiDlTaskPods = (name: string) =>
  instance.get<IResponse<IDlTaskPod[]>>(VERSION + "/recommenddljob/pods", {
    params: {
      name,
    },
  });

// {
//   "p100": {
//       "gpuUtilAvg": 89.072716,
//       "gpuMemoryMaxGB": 3.0564356,
//       "smActiveAvg": 0,
//       "smOccupancyAvg": 0,
//       "fp32ActiveAvg": 0,
//       "dramActiveAvg": 0
//   },
//   "v100": {
//       "gpuUtilAvg": 75.4836,
//       "gpuMemoryMaxGB": 7.706247,
//       "smActiveAvg": 68.70431,
//       "smOccupancyAvg": 61.6621,
//       "fp32ActiveAvg": 3.939799,
//       "dramActiveAvg": 62.314323
//   }
// },
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

// analyze body
// {
//   "runningType": "one-shot",
//   "relationShips": [],
//   "macs": 230006514,
//   "params": 66665211,
//   "batchSize": 64,
//   "vocabularySize": [5000000,10000000,600000],
//   "embeddingDim": [32,32,32]
// }
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
