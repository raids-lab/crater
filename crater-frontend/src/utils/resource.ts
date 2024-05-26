import { z } from "zod";
import { logger } from "./loglevel";
import { showErrorToast } from "./toast";

export interface KubernetesResource {
  cpu?: number | string; // 1
  memory?: string; // "1Gi"
  "nvidia.com/gpu"?: number | string; // "1"
}

export interface KubernetesResourceList {
  [key: string]: string | number;
}

export const AiResourceSchema = z
  .object({
    cpu: z.number().optional(),
    gpu: z.number().optional(),
    memory: z.string().optional(),
    memoryNum: z.number().optional(),
    memoryUnit: z.string().optional(),
  })
  .strict();

export type AiResource = z.infer<typeof AiResourceSchema>;

export const getAiResource = (resource?: KubernetesResource): AiResource => {
  // if resource.cpu is string, parse to int
  // if resource.cpu is number, keep it
  if (!resource) {
    return {
      cpu: 0,
      gpu: 0,
      memory: "0Gi",
      memoryNum: 0,
      memoryUnit: "Gi",
    };
  }
  const cpu = resource.cpu
    ? typeof resource.cpu === "string"
      ? parseInt(resource.cpu)
      : resource.cpu ?? 0
    : 0;
  const gpu = resource["nvidia.com/gpu"]
    ? typeof resource["nvidia.com/gpu"] === "string"
      ? parseInt(resource["nvidia.com/gpu"])
      : resource["nvidia.com/gpu"] ?? 0
    : 0;
  let memory = resource.memory ?? "0Gi";
  let memoryNum = 0;
  const memoryUnit = "Gi";
  // "Ki", "Mi", "Gi" => "Gi"
  if (memory.includes("Ki")) {
    memoryNum = parseInt(memory.replace("Ki", "")) / 1024 / 1024;
    memory = `${memoryNum}Gi`;
  } else if (memory.includes("Mi")) {
    memoryNum = parseInt(memory.replace("Mi", "")) / 1024;
    memory = `${memoryNum}Gi`;
  } else if (memory.includes("Gi") || memory.includes("G")) {
    memoryNum = parseInt(memory.replace("Gi", "").replace("G", ""));
  } else if (memory !== "0") {
    showErrorToast(`Invalid memory ${memory} with unrecognized unit`);
  }

  return {
    cpu,
    gpu,
    memory,
    memoryNum,
    memoryUnit,
  };
};

export const getAiKResource = (
  resource: AiResource,
  gpuTag?: string,
): KubernetesResourceList => {
  // Ensure that cpu, gpu, and memory are not less than 0
  logger.debug("Ai", resource);
  const cpu =
    resource.cpu !== undefined && resource.cpu > -1 ? resource.cpu : undefined;
  const gpu =
    resource.gpu !== undefined && resource.gpu > -1 ? resource.gpu : undefined;
  const memory =
    resource.memory && parseInt(resource.memory) > -1
      ? resource.memory
      : undefined;

  const k8sResource: KubernetesResourceList = {};

  if (cpu !== undefined) {
    k8sResource["cpu"] = cpu;
  }

  if (gpu !== undefined && gpu > 0) {
    k8sResource[gpuTag ?? "nvidia.com/gpu"] = gpu;
  }

  if (memory !== undefined) {
    k8sResource["memory"] = memory;
  }
  logger.debug(k8sResource);
  return k8sResource;
};

export const DlResourceSchema = z
  .object({
    gpu: z.number().optional(),
  })
  .strict();

export type DlResource = z.infer<typeof DlResourceSchema>;

export const getDlResource = (resource: KubernetesResource): DlResource => {
  const gpu =
    typeof resource["nvidia.com/gpu"] === "string"
      ? parseInt(resource["nvidia.com/gpu"])
      : resource["nvidia.com/gpu"] ?? 0;

  return {
    gpu,
  };
};

export const getDlKResource = (resource: DlResource): KubernetesResource => {
  const gpu = resource.gpu;

  return {
    "nvidia.com/gpu": gpu,
  };
};
