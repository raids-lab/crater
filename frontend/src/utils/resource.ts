import { z } from "zod";

export interface KubernetesResource {
  cpu?: number | string; // 1
  memory?: string; // "1Gi"
  "nvidia.com/gpu"?: number | string; // "1"
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

export const getAiResource = (resource: KubernetesResource): AiResource => {
  // if resource.cpu is string, parse to int
  // if resource.cpu is number, keep it
  const cpu =
    typeof resource.cpu === "string"
      ? parseInt(resource.cpu)
      : resource.cpu ?? 0;
  const gpu =
    typeof resource["nvidia.com/gpu"] === "string"
      ? parseInt(resource["nvidia.com/gpu"])
      : resource["nvidia.com/gpu"] ?? 0;
  const memory = resource.memory ?? "0Gi";
  const memoryNum = parseInt(memory.replace("Gi", ""));
  const memoryUnit = "Gi";

  return {
    cpu,
    gpu,
    memory,
    memoryNum,
    memoryUnit,
  };
};

export const getAiKResource = (resource: AiResource): KubernetesResource => {
  // Ensure that cpu, gpu, and memory are not less than 0
  const cpu = resource.cpu && resource.cpu >= 0 ? resource.cpu : undefined;
  const gpu = resource.gpu && resource.gpu >= 0 ? resource.gpu : undefined;
  const memory =
    resource.memory && parseInt(resource.memory) >= 0
      ? resource.memory
      : undefined;

  const k8sResource: KubernetesResource = {};

  if (cpu !== undefined) {
    k8sResource.cpu = cpu;
  }

  if (gpu !== undefined) {
    k8sResource["nvidia.com/gpu"] = gpu;
  }

  if (memory !== undefined) {
    k8sResource.memory = memory;
  }

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
