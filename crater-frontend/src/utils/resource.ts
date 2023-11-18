import { z } from "zod";

export interface KubernetesResource {
  cpu?: number | string; // 1
  memory?: string; // "1Gi"
  "nvidia.com/gpu"?: number | string; // "1"
}

export const ResourceSchema = z
  .object({
    cpu: z.number().optional(),
    gpu: z.number().optional(),
    memory: z.string().optional(),
    memoryNum: z.number().optional(),
    memoryUnit: z.string().optional(),
  })
  .strict();

export type KResource = z.infer<typeof ResourceSchema>;

export const getResource = (resource: KubernetesResource): KResource => {
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

export const getKubernetesResource = (
  resource: KResource,
): KubernetesResource => {
  // if resource.cpu is undefined, then cpu will be undefined
  // if resource.cpu is null, then cpu will be null
  const cpu = resource.cpu;
  const gpu = resource.gpu;
  const memory = resource.memory;

  return {
    cpu,
    "nvidia.com/gpu": gpu,
    memory,
  };
};
