/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { z } from 'zod'

import { MetadataFormType } from '@/components/form/types'

import { V1ResourceList } from './resource'

export const jobNameSchema = z
  .string()
  .min(1, {
    message: '作业名称不能为空',
  })
  .max(40, {
    message: '作业名称最多包含 40 个字符',
  })

export const resourceSchema = z.object({
  cpu: z.number().int().min(0, {
    message: 'CPU 核数不能小于 0',
  }),
  memory: z.number().int().min(0, {
    message: '内存大小不能小于 0',
  }),
  gpu: z
    .object({
      count: z.number().int().min(0, {
        message: '指定的 GPU 卡数不能小于 0',
      }),
      model: z.string().optional(),
    })
    .refine(
      (gpu) => {
        // If a is not null, then b must not be null
        return gpu.count === 0 || (gpu.count > 0 && gpu.model !== null && gpu.model !== undefined)
      },
      {
        message: '加速卡型号不能为空',
        path: ['model'], // The path for the error message
      }
    ),
  network: z
    .object({
      enabled: z.boolean().default(false),
      model: z.string().optional(),
    })
    .refine(
      (rdma) => {
        // If a is not null, then b must not be null
        return !rdma.enabled || (rdma.enabled && rdma.model !== null && rdma.model !== undefined)
      },
      {
        message: 'RDMA 型号不能为空',
        path: ['model'], // The path for the error message
      }
    ),
  vgpu: z.object({
    enabled: z.boolean().default(false),
    models: z
      .array(
        z.object({
          label: z.string().min(1, {
            message: 'vGPU 模型名称不能为空',
          }),
          value: z.number().int().min(0, {
            message: 'vGPU 数量不能小于 0',
          }),
        })
      )
      .optional(),
  }),
})

export type ResourceSchema = z.infer<typeof resourceSchema>

export const defaultResource: ResourceSchema = {
  cpu: 1,
  memory: 2,
  gpu: {
    count: 0,
  },
  network: {
    enabled: false,
  },
  vgpu: {
    enabled: false,
  },
}

export const taskSchema = z.object({
  taskName: z.string().min(1, {
    message: '任务名称不能为空',
  }),
  replicas: z.number().int().min(1, {
    message: '副本数不能小于 1',
  }),
  resource: resourceSchema.required(),
  image: z.string().min(1, {
    message: '容器镜像不能为空',
  }),
  shell: z.string().optional(),
  command: z.string().optional(),
  workingDir: z.string().optional(),
  ports: z.array(
    z.object({
      name: z.string().min(1, {
        message: '端口名称不能为空',
      }),
      port: z.number().int(),
    })
  ),
})

export type TaskSchema = z.infer<typeof taskSchema>

export enum VolumeMountType {
  FileType = 1,
  DataType,
}

export const volumeMountsSchema = z.array(
  z.object({
    type: z.number().int(),
    subPath: z.string().min(1, {
      message: '挂载源不能为空',
    }),
    datasetID: z.number().int().nonnegative().optional(),
    mountPath: z
      .string()
      .min(1, {
        message: '挂载到容器中的路径不能为空',
      })
      .refine((value) => value.startsWith('/'), {
        message: '路径需以单个斜杠 `/` 开头',
      })
      .refine((value) => !value.includes('..'), {
        message: '禁止使用相对路径 `..`',
      })
      .refine((value) => !value.includes('//'), {
        message: '避免使用多个连续的斜杠 `//`',
      })
      .refine((value) => value !== '/', {
        message: '禁止挂载到根目录 `/`',
      }),
  })
)

export type VolumeMountsSchema = z.infer<typeof volumeMountsSchema>

export const envsSchema = z.array(
  z.object({
    name: z.string().min(1, {
      message: '环境变量名不能为空',
    }),
    value: z.string().min(1, {
      message: '环境变量值不能为空',
    }),
  })
)

export type EnvsSchema = z.infer<typeof envsSchema>

export const forwardsSchema = z.array(
  z.object({
    name: z
      .string()
      .min(1)
      .max(20)
      .regex(/^[a-z]+$/, {
        message: '只能包含小写字母',
      }),
    port: z.number().int().positive(),
  })
)

export type ForwardsSchema = z.infer<typeof forwardsSchema>

export const nodeSelectorSchema = z
  .object({
    enable: z.boolean(),
    nodeName: z.string().optional(),
  })
  .refine(
    (observability) => {
      return (
        !observability.enable ||
        (observability.enable &&
          observability.nodeName !== null &&
          observability.nodeName !== undefined)
      )
    },
    {
      message: '节点名称不能为空',
      path: ['nodeName'],
    }
  )

export type NodeSelectorSchema = z.infer<typeof nodeSelectorSchema>

export interface JobSubmitJson<T> {
  version: string
  type: string
  data: T
}

// exportToJsonFile convert T to JSON and save it to file and download it.
export const exportToJsonFile = (data: JobSubmitJson<unknown>, filename = 'data.json') => {
  const json = JSON.stringify(data, null, 2)
  const blob = new Blob([json], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export const exportToJsonString = (metadata: MetadataFormType, data: unknown): string => {
  const jobInfo: JobSubmitJson<unknown> = {
    version: metadata.version,
    type: metadata.type,
    data,
  }
  return JSON.stringify(jobInfo, null, 2)
}

// usage: importFromJson<FormData>(version, type, file)
export const importFromJsonFile = async <T>(
  version: string,
  type: string,
  file?: File
): Promise<T> => {
  if (!file) {
    throw new Error('无效的配置文件')
  }
  const text = await file.text()
  const jobInfo = JSON.parse(text) as JobSubmitJson<T>
  if (jobInfo.version !== version) {
    throw new Error(
      `当前配置版本为 ${version}，导入配置版本为 ${jobInfo.version}，无法导入，请手动创建新配置`
    )
  } else if (jobInfo.type !== type) {
    throw new Error(
      `当前配置类型为 ${type}，导入配置类型为 ${jobInfo.type}，无法导入，请手动创建新配置`
    )
  }
  return jobInfo.data
}

export const importFromJsonString = <T>(metadata: MetadataFormType, text: string): T => {
  const jobInfo = JSON.parse(text) as JobSubmitJson<T>
  if (jobInfo.version !== metadata.version) {
    throw new Error(
      `当前配置版本为 ${metadata.version}，导入配置版本为 ${jobInfo.version}，无法导入，请手动创建新配置`
    )
  } else if (jobInfo.type !== metadata.type) {
    throw new Error(
      `当前配置类型为 ${metadata.type}，导入配置类型为 ${jobInfo.type}，无法导入，请手动创建新配置`
    )
  }
  return jobInfo.data
}

export const convertToResourceList = (resource: ResourceSchema): V1ResourceList => {
  const k8sResource: V1ResourceList = {
    cpu: `${resource.cpu}`,
    memory: `${resource.memory}Gi`,
  }
  if (resource.gpu.model && resource.gpu.count > 0) {
    if (resource.network.enabled && resource.network.model) {
      k8sResource[resource.network.model] = '1'
    }
    if (resource.vgpu.enabled && resource.vgpu.models && resource.vgpu.models?.length > 0) {
      resource.vgpu.models.forEach((model) => {
        k8sResource[model.label] = `${model.value}`
      })
    }
    k8sResource[resource.gpu.model] = `${resource.gpu.count}`
  }
  return k8sResource
}
