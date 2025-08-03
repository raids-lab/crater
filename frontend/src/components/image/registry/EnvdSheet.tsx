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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CircleHelpIcon, PackagePlusIcon } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

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
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import LoadableButton from '@/components/button/loadable-button'
import Combobox from '@/components/form/Combobox'
import FormExportButton from '@/components/form/FormExportButton'
import FormImportButton from '@/components/form/FormImportButton'
import FormLabelMust from '@/components/form/FormLabelMust'
import { ImageSettingsFormCard } from '@/components/form/ImageSettingsFormCard'
import { TagsInput } from '@/components/form/TagsInput'
import { MetadataFormEnvdAdvanced } from '@/components/form/types'
import SandwichSheet, { SandwichLayout, SandwichSheetProps } from '@/components/sheet/SandwichSheet'

import {
  FetchAllUniqueImageTagObjects,
  ImagePackSource,
  apiGetCudaBaseImages,
  apiUserCreateByEnvd,
  imageNameRegex,
  imageTagRegex,
} from '@/services/api/imagepack'

import { useImageTemplateLoader } from '@/hooks/useTemplateLoader'

import { exportToJsonString } from '@/utils/form'

const envdFormSchema = z.object({
  python: z.string().min(1, 'Python version is required'),
  base: z.string().min(1, 'CUDA version is required'),
  description: z.string().min(1, '请为镜像添加描述'),
  aptPackages: z.string().optional(),
  requirements: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          try {
            v.split('\n').forEach((line) => {
              if (line.trim().startsWith('#')) {
                return
              }
              if (!line.trim()) {
                return
              }
              // relation:
              // ==：等于
              // >：大于版本
              // >=：大于等于
              // <：小于版本
              // <=：小于等于版本
              // ~=：兼容版本

              // 基于 relation 将每一行的内容进行分割，分割后的内容为：name, relation, version
              // 可以只有 name
              const regex = /([a-zA-Z0-9_]+)([<>=!~]+)?([a-zA-Z0-9_.]+)?/
              const match = line.match(regex)
              if (!match) {
                throw new Error('Invalid requirement format')
              }
              if (match.length < 2) {
                throw new Error('Invalid requirement format')
              }
            })
          } catch {
            return false
          }
        }
        return true
      },
      {
        message: 'requirements.txt 文件格式无效',
      }
    ),
  enableJupyter: z.boolean().optional(),
  enableZsh: z.boolean().optional(),
  imageName: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          return imageNameRegex.test(v)
        } else {
          return true
        }
      },
      {
        message: '仅允许小写字母、数字、. _ -，且不能以分隔符开头/结尾',
      }
    )
    .refine((v) => !v || v.includes('envd'), {
      message: "名称需包含 'envd' ",
    }),
  imageTag: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          return imageTagRegex.test(v)
        } else {
          return true
        }
      },
      {
        message: '仅允许字母、数字、_ . + -，且不能以 . 或 - 开头/结尾',
      }
    )
    .refine((value) => value !== 'latest', {
      message: "镜像标签不能为: 'latest'",
    }),
  tags: z
    .array(
      z.object({
        value: z.string(),
      })
    )
    .optional(),
})

export type EnvdFormValues = z.infer<typeof envdFormSchema>

interface EnvdSheetContentProps {
  closeSheet: () => void
  imagePackName: string
  setImagePackName: (imagePackName: string) => void
}

function EnvdSheetContent({ closeSheet, imagePackName, setImagePackName }: EnvdSheetContentProps) {
  const queryClient = useQueryClient()

  const form = useForm<EnvdFormValues>({
    resolver: zodResolver(envdFormSchema),
    defaultValues: {
      description: '',
      imageName: '',
      imageTag: '',
      tags: [{ value: 'Jupyter' }],
      enableJupyter: true,
      enableZsh: true,
    },
  })

  const { mutate: submitDockerfileSheet, isPending } = useMutation({
    mutationFn: (values: EnvdFormValues) =>
      apiUserCreateByEnvd({
        description: values.description,
        envd: generateBuildScript(
          values.base,
          values.python,
          values.aptPackages
            ?.split(' ')
            .map((item) => item.trim())
            .filter(Boolean),
          values.requirements
            ?.split('\n')
            .map((item) => item.trim())
            .filter(Boolean),
          values.enableJupyter ?? false,
          values.enableZsh ?? false
        ),
        name: values.imageName ?? '',
        tag: values.imageTag ?? '',
        python: values.python,
        base: cudaBaseImages.find((image) => image.value === values.base)?.imageLabel ?? '',
        tags: values.tags?.map((item) => item.value) ?? [],
        template: exportToJsonString(MetadataFormEnvdAdvanced, values),
        buildSource: ImagePackSource.EnvdAdvanced,
      }),
    onSuccess: async () => {
      await new Promise((resolve) => setTimeout(resolve, 500)).then(() =>
        queryClient.invalidateQueries({ queryKey: ['imagepack', 'list'] })
      )
      closeSheet()
      toast.success(`镜像开始制作，请在下方列表中查看制作状态`)
    },
  })

  const { data: allImageTags } = useQuery({
    queryKey: ['allImageTags'],
    queryFn: FetchAllUniqueImageTagObjects,
  })

  const onSubmit = (values: EnvdFormValues) => {
    submitDockerfileSheet(values)
  }

  const {
    data: cudaBaseImages = [],
    error: cudaBaseImagesError,
    isLoading: isCudaLoading,
  } = useQuery({
    queryKey: ['cudaBaseImages'],
    queryFn: apiGetCudaBaseImages,
    select: (res) => {
      // eslint-disable-next-line no-console
      console.log('CUDA Base Images API Response:', res)
      return res.data.cudaBaseImages.map((item) => ({
        label: item.label,
        value: item.value,
        imageLabel: item.imageLabel,
      }))
    },
  })

  useImageTemplateLoader({
    form: form,
    metadata: MetadataFormEnvdAdvanced,
    imagePackName: imagePackName,
    setImagePackName: setImagePackName,
  })

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SandwichLayout
          footer={
            <>
              <FormImportButton metadata={MetadataFormEnvdAdvanced} form={form} />
              <FormExportButton metadata={MetadataFormEnvdAdvanced} form={form} />
              <LoadableButton isLoading={isPending} isLoadingText="正在提交" type="submit">
                <PackagePlusIcon />
                开始制作
              </LoadableButton>
            </>
          }
        >
          <FormField
            control={form.control}
            name="description"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  描述
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormDescription>
                  关于此镜像的简短描述，如包含的软件版本、用途等，将作为镜像标识显示
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className="grid grid-cols-2 gap-4">
            <FormField
              control={form.control}
              name="python"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    Python 版本
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Combobox
                      items={PYTHON_VERSIONS.map((version) => ({
                        label: version,
                        value: version,
                      }))}
                      current={field.value}
                      handleSelect={(value) => field.onChange(value)}
                      formTitle="Python版本"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="base"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    CUDA 版本
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Combobox
                      items={cudaBaseImages}
                      current={field.value}
                      handleSelect={(value) => field.onChange(value)}
                      formTitle="CUDA版本"
                      disabled={isCudaLoading}
                    />
                  </FormControl>
                  {cudaBaseImagesError && (
                    <FormDescription className="text-destructive">
                      获取 CUDA 版本失败，请联系管理员
                    </FormDescription>
                  )}
                  {isCudaLoading && <FormDescription>正在加载 CUDA 版本...</FormDescription>}
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <TagsInput
            form={form}
            tagsPath="tags"
            label={`镜像标签`}
            description={`为镜像添加标签，以便分类和搜索`}
            customTags={allImageTags}
          />
          <FormField
            control={form.control}
            name="aptPackages"
            render={({ field }) => (
              <FormItem>
                <FormLabel>APT Packages</FormLabel>
                <FormControl>
                  <Textarea placeholder="git curl" className="h-24 font-mono" {...field} />
                </FormControl>
                <FormDescription>输入要安装的 APT 包，使用空格分隔多个包</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="requirements"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Python 依赖</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder={`transformers>=4.46.3
diffusers==0.31.0`}
                    className="h-24 font-mono"
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  请粘贴 requirements.txt 文件的内容，以便安装所需的 Python 包
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className="flex flex-col gap-1.5">
            <FormField
              control={form.control}
              name="enableJupyter"
              render={({ field }) => (
                <FormItem>
                  <div className="flex flex-row items-center justify-between space-y-0 space-x-0">
                    <FormLabel>
                      启用 Jupyter
                      <TooltipProvider delayDuration={100}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>启用 Jupyter Notebook 支持，允许交互式开发和数据分析</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </FormLabel>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={(value) => field.onChange(value)}
                      />
                    </FormControl>
                  </div>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="enableZsh"
              render={({ field }) => (
                <FormItem>
                  <div className="flex flex-row items-center justify-between space-y-0 space-x-0">
                    <FormLabel>
                      启用 ZSH Shell
                      <TooltipProvider delayDuration={100}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>安装并配置 ZSH 作为默认 Shell，包含 Oh My Zsh 和常用插件</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </FormLabel>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={(value) => field.onChange(value)}
                      />
                    </FormControl>
                  </div>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <ImageSettingsFormCard form={form} imageNamePath="imageName" imageTagPath="imageTag" />
        </SandwichLayout>
      </form>
    </Form>
  )
}

interface EnvdSheetProps extends SandwichSheetProps {
  closeSheet: () => void
  imagePackName?: string
  setImagePackName: (imagePackName: string) => void
}

export function EnvdSheet({
  closeSheet,
  imagePackName = '',
  setImagePackName,
  ...props
}: EnvdSheetProps) {
  return (
    <SandwichSheet {...props}>
      <EnvdSheetContent
        closeSheet={closeSheet}
        imagePackName={imagePackName}
        setImagePackName={setImagePackName}
      />
    </SandwichSheet>
  )
}

const PYTHON_VERSIONS = ['3.13', '3.12', '3.11', '3.10', '3.9', '3.8', '3.7', '3.6']

const generateBuildScript = (
  baseImage: string,
  pythonVersion: string = '3.12',
  extraAptPackages: string[] = [],
  extraPythonPackages: string[] = [],
  enableJupyter: boolean = true,
  enableZsh: boolean = true
) => {
  // Combine all APT packages into one array
  const aptPackages = [
    'openssh-server',
    'build-essential',
    'iputils-ping',
    'net-tools',
    'htop',
    'tree',
    'tzdata',
    ...extraAptPackages,
  ]

  // Build the envd script
  let script = `# syntax=v1

def build():
    base(image="${baseImage}", dev=True)
    install.python(version="${pythonVersion}")
    install.apt_packages([${aptPackages.map((item) => `"${item}"`).join(', ')}])
    config.repo(
        url="https://github.com/raids-lab/crater",
        description="Crater",
    )
    config.pip_index(url="https://pypi.tuna.tsinghua.edu.cn/simple")`

  // Add Python packages if any
  if (extraPythonPackages.length > 0) {
    script += `
    install.python_packages(name=[${extraPythonPackages.map((item) => `"${item}"`).join(', ')}])`
  }

  if (enableJupyter && enableZsh) {
    script += `
    run(commands=[
      "chsh -s /bin/zsh root;",
      "git clone --depth 1 https://gitee.com/mirrors/oh-my-zsh.git;",
      "ZSH=\\"/usr/share/.oh-my-zsh\\" CHSH=\\"no\\" RUNZSH=\\"no\\" REMOTE=https://gitee.com/mirrors/oh-my-zsh.git sh ./oh-my-zsh/tools/install.sh;",
      "chmod a+rx /usr/share/.oh-my-zsh/oh-my-zsh.sh;",
      "rm -rf ./oh-my-zsh;",
      "git clone --depth=1 https://gitee.com/mirrors/zsh-syntax-highlighting.git /usr/share/.oh-my-zsh/custom/plugins/zsh-syntax-highlighting;",
      "git clone --depth=1 https://gitee.com/mirrors/zsh-autosuggestions.git /usr/share/.oh-my-zsh/custom/plugins/zsh-autosuggestions;",
      "echo \\"export skip_global_compinit=1\\" >> /etc/zsh/zshenv;",
      "echo \\"export ZSH=\\\\\\"/usr/share/.oh-my-zsh\\\\\\"\\" >> /etc/zsh/zshrc;",
      "echo \\"plugins=(git extract sudo jsontools colored-man-pages zsh-autosuggestions zsh-syntax-highlighting)\\" >> /etc/zsh/zshrc;",
      "echo \\"ZSH_THEME=\\\\\\"robbyrussell\\\\\\"\\" >> /etc/zsh/zshrc;",
      "echo \\"export ZSH_COMPDUMP=\\\\$ZSH/cache/.zcompdump-\\\\$HOST\\" >> /etc/zsh/zshrc;",
      "mkdir -p /etc/jupyter;",
      "echo \\"c.ServerApp.terminado_settings = {\\\\\\"shell_command\\\\\\": [\\\\\\"/bin/zsh\\\\\\"]}\\" >> /etc/jupyter/jupyter_server_config.py;",
      "echo \\"source \\\\$ZSH/oh-my-zsh.sh\\" >> /etc/zsh/zshrc;",
      "echo \\"zstyle \\\\\\":omz:update\\\\\\" mode disabled\\" >> /etc/zsh/zshrc;",
    ])`
  }

  // Add Jupyter config if enabled
  if (enableJupyter) {
    script += `
    config.jupyter()`
  }

  return script
}
