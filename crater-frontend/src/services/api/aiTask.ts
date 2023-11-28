import { KubernetesResource } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { showErrorToast } from "@/utils/toast";

export interface IAiTask {
  id: number;
  taskName: string;
  userName: string;
  namespace: string;
  taskType: string;
  createdAt: string;
  updatedAt: string;
  admittedAt: string;
  startedAt: string;
  image: string;
  resourceRequest: string;
  workingDir: string;
  ShareDirs: string[];
  command: string;
  args: string;
  slo: number;
  status: string;
  isDeleted: false;
  profileStatus: number;
  profileStat: string;
  estimatedTime: number;
  scheduleInfo: string;
}

// profileStat: {
//   gpu_util_avg: 0.97, //上限1
//   gpu_util_max: 0.98, //上限1
//   gpu_util_std: 0.0031622776,
//   sm_util_avg: 0.7899093, //上限1
//   sm_util_max: 0.798531, //上限1
//   sm_util_std: 0.0085214125,
//   sm_occupancy_avg: 0.24534787, //上限1
//   sm_occupancy_max: 0.250931,
//   sm_occupancy_std: 0.0044290754,
//   dram_util_avg: 0.24091028, //上限1
//   dram_util_max: 0.247308,
//   dram_util_std: 0.0039772647,
//   mem_copy_util_avg: 0.3405, //上限1
//   mem_copy_util_max: 0.35,
//   mem_copy_util_std: 0.0028431204,
//   pcie_tx_avg: 92.72305, // 展示数值 MB/s
//   pcie_tx_max: 109.68077,
//   pcie_rx_avg: 717.1082, // 展示数值 MB/s
//   pcie_rx_max: 900995800,
//   cpu_usage_avg: 1.3134052, //展示数值
//   gpu_mem_max: 10007, //展示单位MB，上限是32768
//   cpu_mem_max: 7325.039, // 展示单位MB，不用进度条
// };
interface ProfileStat {
  gpu_util_avg: number; //上限1
  gpu_util_max: number; //上限1
  gpu_util_std: number;
  sm_util_avg: number; //上限1
  sm_util_max: number; //上限1
  sm_util_std: number;
  sm_occupancy_avg: number; //上限1
  sm_occupancy_max: number;
  sm_occupancy_std: number;
  dram_util_avg: number; //上限1
  dram_util_max: number;
  dram_util_std: number;
  mem_copy_util_avg: number; //上限1
  mem_copy_util_max: number;
  mem_copy_util_std: number;
  pcie_tx_avg: number; // 展示数值 MB/s
  pcie_tx_max: number;
  pcie_rx_avg: number; // 展示数值 MB/s
  pcie_rx_max: number;
  cpu_usage_avg: number; //展示数值
  gpu_mem_max: number; //展示单位MB，上限是32768
  cpu_mem_max: number; // 展示单位MB，不用进度条
}

export interface AiTask {
  id: number;
  taskName: string;
  userName: string;
  namespace: string;
  taskType: string;
  createdAt: string;
  updatedAt: string;
  admittedAt: string;
  startedAt: string;
  image: string;
  resourceRequest: KubernetesResource;
  workingDir: string;
  ShareDirs: string[];
  command: string;
  args: string;
  slo: number;
  status: string;
  isDeleted: false;
  profileStatus: number;
  profileStat?: ProfileStat;
  estimatedTime: number;
  scheduleInfo: string;
}

export const convertAiTask = (task: IAiTask): AiTask => {
  try {
    const aiTaskINfo: AiTask = {
      ...task,
      resourceRequest: JSON.parse(task.resourceRequest) as KubernetesResource,
      profileStat:
        task.profileStat === ""
          ? undefined
          : (JSON.parse(task.profileStat) as ProfileStat),
    };
    return aiTaskINfo;
  } catch (e) {
    showErrorToast("任务信息解析失败", e);
    return task as unknown as AiTask;
  }
};

export interface ITaskCreate {
  taskName: string;
  slo: number;
  taskType: string;
  resourceRequest: KubernetesResource;
  image: string;
  workingDir: string;
  shareDirs: {
    [key: string]: string;
  };
  command: string;
  args: {
    [key: string]: string;
  };
}

export const apiAiTaskCreate = async (task: ITaskCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/aitask/create",
    task,
  );
  return response.data;
};

export const apiAiTaskDelete = async (taskID: number) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/aitask/delete",
    {
      taskID,
    },
  );
  return response.data;
};

export const apiAiTaskList = () =>
  instance.get<
    IResponse<{
      Tasks: IAiTask[];
    }>
  >(VERSION + "/aitask/list");

export const apiAiTaskQuota = () =>
  instance.get<
    IResponse<{
      hard: KubernetesResource;
      hardUsed: KubernetesResource;
      softUsed: KubernetesResource;
    }>
  >(VERSION + "/aitask/getQuota");

export const apiAiTaskGet = (taskID: number) =>
  instance.get<IResponse<IAiTask>>(VERSION + "/aitask/get", {
    params: {
      taskID,
    },
  });

//   path: /v1/sharedir/list
//   method: GET
//   response
// {
// "data": [
//     "dnn-train-data",
//     "jupyterhub-shared-volume"
// ],
// "error": "",
// "status": true
// }
export const apiAiTaskShareDirList = () =>
  instance.get<IResponse<string[]>>(VERSION + "/sharedir/list");
