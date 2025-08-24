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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, useRouter } from '@tanstack/react-router'
import { t } from 'i18next'
import { useAtomValue } from 'jotai'
import { CirclePlus, LayoutGridIcon } from 'lucide-react'
import { useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import LoadableButton from '@/components/button/loadable-button'
import { VolumeMountsCard } from '@/components/form/DataMountFormField'
import { EnvFormCard } from '@/components/form/EnvFormField'
import FormExportButton from '@/components/form/FormExportButton'
import FormImportButton from '@/components/form/FormImportButton'
import FormLabelMust from '@/components/form/FormLabelMust'
import { ForwardFormCard } from '@/components/form/ForwardFormField'
import { ImageFormField } from '@/components/form/ImageFormField'
import { OtherOptionsFormCard } from '@/components/form/OtherOptionsFormField'
import { ResourceFormFields } from '@/components/form/ResourceFormField'
import { TemplateInfo } from '@/components/form/TemplateInfo'
import { MetadataFormJupyter } from '@/components/form/types'
import { PublishConfigForm, publishValidateSearch } from '@/components/job/publish'
import CardTitle from '@/components/label/CardTitle'
import PageTitle from '@/components/layout/page-title'

import { apiJupyterCreate } from '@/services/api/vcjob'

import {
  VolumeMountType,
  convertToResourceList,
  defaultResource,
  envsSchema,
  exportToJsonString,
  forwardsSchema,
  jobNameSchema,
  nodeSelectorSchema,
  taskSchema,
  volumeMountsSchema,
} from '@/utils/form'
import { atomUserInfo } from '@/utils/store'
import { configFeatureFlags } from '@/utils/store/config'

export const Route = createFileRoute('/portal/jobs/new/jupyter-job')({
  validateSearch: publishValidateSearch,
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('jobs.new.jupyterJob'),
    }
  },
})

const JupyterMarkdown = `Jupyter 为用户提供交互式的 Web 实验环境，可用于代码调试等场景。

## 排队规则

1. Jupyter 作业排队超过 5 分钟未有资源可用时，作业会被自动取消。这种情况下，用户可以尝试调整资源配置，或者选择 [自定义作业](/portal/job/batch) 进行提交。

## 运行规则

2. 如果申请了 GPU 资源，当过去 2 个小时 GPU 利用率为 0，我们将尝试发送告警信息给用户，建议用户检查作业是否正常运行。若此后半小时 GPU 利用率仍为 0，**系统将释放作业占用的资源**。
3. 当作业运行超过 4 天，我们将尝试发送告警信息给用户，提醒用户作业运行时间过长；若此后一天内用户未联系管理员说明情况并锁定作业，**系统将释放作业占用的资源**。
`

const formSchema = z.object({
  jobName: jobNameSchema,
  task: taskSchema,
  envs: envsSchema,
  volumeMounts: volumeMountsSchema,
  nodeSelector: nodeSelectorSchema,
  alertEnabled: z.boolean().default(true),
  forwards: forwardsSchema,
})

type FormSchema = z.infer<typeof formSchema>

const dataProcessor = (data: FormSchema) => {
  // 如果需要在不改变 MetadataFormJupyter 版本号的情况下，保持兼容性
  // 可以在这里进行数据转换

  // if rdma is enabled, set it to false
  if (!data.task.resource.network) {
    data.task.resource.network = {
      enabled: false,
      model: undefined,
    }
  }
  if (!data.task.resource.vgpu) {
    data.task.resource.vgpu = {
      enabled: false,
    }
  }
  return data
}

function RouteComponent() {
  const searchParams = Route.useSearch()
  const [envOpen, setEnvOpen] = useState(false)
  const [otherOpen, setOtherOpen] = useState(true)
  const queryClient = useQueryClient()
  const user = useAtomValue(atomUserInfo)
  const flags = useAtomValue(configFeatureFlags)
  const router = useRouter()

  const { mutate: createTask, isPending } = useMutation({
    mutationFn: (values: FormSchema) => {
      return apiJupyterCreate({
        name: values.jobName,
        resource: convertToResourceList(values.task.resource),
        image: values.task.image,
        volumeMounts: values.volumeMounts,
        envs: values.envs,
        alertEnabled: values.alertEnabled,
        selectors: values.nodeSelector.enable
          ? [
              {
                key: 'kubernetes.io/hostname',
                operator: 'In',
                values: [`${values.nodeSelector.nodeName}`],
              },
            ]
          : undefined,
        template: exportToJsonString(MetadataFormJupyter, values),
        forwards: values.forwards,
      })
    },
    onSuccess: async (_, { jobName }) => {
      await new Promise((resolve) => setTimeout(resolve, 500)).then(() =>
        queryClient.invalidateQueries({ queryKey: ['job'] })
      )
      toast.success(`作业 ${jobName} 创建成功`)
      router.history.back()
    },
  })

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      jobName: '',
      task: {
        taskName: 'training',
        replicas: 1,
        resource: defaultResource,
        image: '',
        shell: '',
        command: '',
        workingDir: `/home/${user?.name}`,
        ports: [],
      },
      volumeMounts: [
        {
          type: VolumeMountType.FileType,
          subPath: 'user',
          mountPath: `/home/${user?.name}`,
        },
      ],
      forwards: [],
      envs: [],
      alertEnabled: true,
      nodeSelector: {
        enable: false,
      },
    },
  })

  useFieldArray<FormSchema>({
    name: 'forwards',
    control: form.control,
  })

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    if (isPending) {
      toast.info('请勿重复提交')
      return
    }
    if (
      flags.alertLowCPURequest &&
      values.task.resource.gpu.count > 0 &&
      values.task.resource.cpu <= 2 &&
      values.task.resource.memory <= 4
    ) {
      form.setError('task.resource.gpu.model', {
        type: 'manual',
        message: '建议结合节点资源分配情况，妥善调整 CPU 和内存资源申请，避免作业被 OOM Kill',
      })
      form.setError('task.resource.cpu', {
        type: 'manual',
        message: '请增加 CPU 核数',
      })
      form.setError('task.resource.memory', {
        type: 'manual',
        message: '请增加内存大小',
      })
      return
    }
    createTask(values)
  }

  return (
    <>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="grid items-start gap-4 md:gap-6 lg:grid-cols-3"
        >
          <PageTitle
            title="新建 Jupyter Lab"
            description="提供交互式的 Web 实验环境，可用于代码调试"
            className="lg:col-span-3"
            tipContent={`版本 ${MetadataFormJupyter.version}`}
          >
            <div className="items-centor flex w-fit flex-row justify-end gap-3">
              <FormImportButton
                metadata={MetadataFormJupyter}
                dataProcessor={dataProcessor}
                form={form}
                afterImport={(data) => {
                  if (data.envs.length > 0) {
                    setEnvOpen(true)
                  }
                  if (data.nodeSelector.enable) {
                    setOtherOpen(true)
                  }
                }}
              />
              <FormExportButton metadata={MetadataFormJupyter} form={form} />
              <PublishConfigForm
                config={MetadataFormJupyter}
                configform={form}
                fromTemplate={searchParams.fromTemplate}
              />
              <LoadableButton isLoading={isPending} isLoadingText="提交作业" type="submit">
                <CirclePlus className="size-4" />
                提交作业
              </LoadableButton>
            </div>
          </PageTitle>
          <div className="flex flex-col gap-4 md:gap-6 lg:col-span-2">
            <Card>
              <CardHeader>
                <CardTitle icon={LayoutGridIcon}>基本设置</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-5">
                <FormField
                  control={form.control}
                  name="jobName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        作业名称
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input {...field} autoFocus={true} />
                      </FormControl>
                      <FormDescription>名称可重复，最多包含 40 个字符</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <ResourceFormFields
                  form={form}
                  cpuPath="task.resource.cpu"
                  memoryPath="task.resource.memory"
                  gpuCountPath="task.resource.gpu.count"
                  gpuModelPath="task.resource.gpu.model"
                  rdmaPath={{
                    rdmaEnabled: 'task.resource.network.enabled',
                    rdmaLabel: 'task.resource.network.model',
                  }}
                  vgpuPath={{
                    vgpuEnabled: 'task.resource.vgpu.enabled',
                    vgpuModels: 'task.resource.vgpu.models',
                  }}
                />
                <ImageFormField form={form} name="task.image" />
              </CardContent>
            </Card>
            <TemplateInfo
              form={form}
              metadata={MetadataFormJupyter}
              searchParams={searchParams}
              uiStateUpdaters={[
                {
                  condition: (data) => data.envs.length > 0,
                  setter: setEnvOpen,
                  value: true,
                },
                {
                  condition: (data) => data.nodeSelector.enable || data.alertEnabled,
                  setter: setOtherOpen,
                  value: true,
                },
              ]}
              dataProcessor={dataProcessor}
              defaultMarkdown={JupyterMarkdown}
            />
          </div>
          <div className="flex flex-col gap-4 md:gap-6">
            <VolumeMountsCard form={form} />
            <ForwardFormCard form={form} />
            <EnvFormCard form={form} open={envOpen} setOpen={setEnvOpen} />
            <OtherOptionsFormCard
              form={form}
              alertEnabledPath="alertEnabled"
              nodeSelectorEnablePath="nodeSelector.enable"
              nodeSelectorNodeNamePath="nodeSelector.nodeName"
              open={otherOpen}
              setOpen={setOtherOpen}
            />
          </div>
        </form>
      </Form>
    </>
  )
}
