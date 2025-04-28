import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import { useNavigate } from "react-router-dom";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  apiJupyterCreate,
  apiJobTemplate,
  JobType,
} from "@/services/api/vcjob";
import { toast } from "sonner";
import { CirclePlus } from "lucide-react";
import FormLabelMust from "@/components/form/FormLabelMust";
import {
  convertToResourceList,
  defaultResource,
  envsSchema,
  exportToJsonString,
  forwardsSchema,
  jobNameSchema,
  nodeSelectorSchema,
  taskSchema,
  volumeMountsSchema,
  VolumeMountType,
} from "@/utils/form";
import { useEffect, useState } from "react";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { MetadataFormJupyterEmias } from "@/components/form/types";
import LoadableButton from "@/components/custom/LoadableButton";
import PageTitle from "@/components/layout/PageTitle";
import { PublishConfigForm } from "./Publish";
import { ImageFormField } from "@/components/form/ImageFormField";
import { VolumeMountsCard } from "@/components/form/DataMountFormField";
import { ResourceFormFields } from "@/components/form/ResourceFormField";
import { TemplateInfo } from "@/components/form/TemplateInfo";
import { OtherOptionsFormCard } from "@/components/form/OtherOptionsFormField";
import { EnvFormCard } from "@/components/form/EnvFormField";
import { ForwardFormCard } from "@/components/form/ForwardFormField";
import FormExportButton from "@/components/form/FormExportButton";
import FormImportButton from "@/components/form/FormImportButton";

const JupyterMarkdown = `Jupyter 为用户提供交互式的 Web 实验环境，可用于代码调试等场景。

## 排队规则

1. Jupyter 作业排队超过 5 分钟未有资源可用时，作业会被自动取消。这种情况下，用户可以尝试调整资源配置，或者选择 [自定义作业](/portal/job/batch) 进行提交。

## 运行规则

2. 如果申请了 GPU 资源，当过去 2 个小时 GPU 利用率为 0，我们将尝试发送告警信息给用户，建议用户检查作业是否正常运行。若此后半小时 GPU 利用率仍为 0，**系统将释放作业占用的资源**。
3. 当作业运行超过 4 天，我们将尝试发送告警信息给用户，提醒用户作业运行时间过长；若此后一天内用户未联系管理员说明情况并锁定作业，**系统将释放作业占用的资源**。
`;

const formSchema = z.object({
  jobName: jobNameSchema,
  task: taskSchema,
  envs: envsSchema,
  volumeMounts: volumeMountsSchema,
  nodeSelector: nodeSelectorSchema,
  alertEnabled: z.boolean().default(true),
  forwards: forwardsSchema,
});

type FormSchema = z.infer<typeof formSchema>;

const dataProcessor = (data: FormSchema) => {
  // if rdma is enabled, set it to false
  if (data.task.resource.network) {
    data.task.resource.network.enabled = false;
    data.task.resource.network.model = undefined;
  }
  if (!data.task.resource.network) {
    data.task.resource.network = {
      enabled: false,
      model: undefined,
    };
  }
  return data;
};

export const Component = () => {
  const [envOpen, setEnvOpen] = useState<boolean>(false);
  const [otherOpen, setOtherOpen] = useState<boolean>(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAtomValue(globalUserInfo);

  const { mutate: createTask, isPending } = useMutation({
    mutationFn: (values: FormSchema) => {
      return apiJupyterCreate({
        name: values.jobName,
        resource: convertToResourceList(values.task.resource),
        image: values.task.image,
        volumeMounts: values.volumeMounts,
        envs: values.envs,
        forwards: values.forwards,
        alertEnabled: values.alertEnabled,
        selectors: values.nodeSelector.enable
          ? [
              {
                key: "kubernetes.io/hostname",
                operator: "In",
                values: [`${values.nodeSelector.nodeName}`],
              },
            ]
          : undefined,
        template: exportToJsonString(MetadataFormJupyterEmias, values),
      });
    },
    onSuccess: async (_, { jobName }) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["job"] }),
        queryClient.invalidateQueries({ queryKey: ["context", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
      toast.success(`作业 ${jobName} 创建成功`);
      navigate(-1);
    },
  });

  const { mutate: fetchJobTemplate } = useMutation({
    mutationFn: (jobName: string) => apiJobTemplate(jobName),
    onSuccess: (response) => {
      const jobInfo = JSON.parse(response.data.data);

      const newData = {
        jobName: jobInfo.data.taskname,
        task: {
          resource: {
            cpu: jobInfo.data.cpu,
            memory: jobInfo.data.memory,
            gpu: jobInfo.data.gpu,
          },
          image: jobInfo.data.image,
        },
        envs: jobInfo.data.envs || [],
        volumeMounts: jobInfo.data.volumeMounts || [],
        nodeSelector: jobInfo.data.nodeSelector || { enable: false },
        alertEnabled:
          jobInfo.data.alertEnabled !== undefined
            ? jobInfo.data.alertEnabled
            : true,
        forwards: jobInfo.data.forwards || [],
      };

      form.reset(newData);

      if (newData.envs.length > 0) {
        setEnvOpen(true);
      }
      if (newData.nodeSelector.enable) {
        setOtherOpen(true);
      }
    },
    onError: () => {
      toast.error(`解析错误，导入配置失败`);
    },
  });

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const fromJob = params.get("fromJob");
    if (fromJob) {
      fetchJobTemplate(fromJob);
    }
  }, [fetchJobTemplate]);

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      jobName: "",
      task: {
        taskName: "training",
        replicas: 1,
        resource: defaultResource,
        image: "",
        shell: "",
        command: "",
        workingDir: `/home/${user.name}`,
        ports: [],
      },
      volumeMounts: [
        {
          type: VolumeMountType.FileType,
          subPath: `user`,
          mountPath: `/home/${user.name}`,
        },
      ],
      envs: [],
      forwards: [],
      nodeSelector: {
        enable: false,
      },
      alertEnabled: true,
    },
  });

  const onSubmit = (values: FormSchema) => {
    if (isPending) {
      toast.info("请勿重复提交");
      return;
    }
    createTask(values);
  };

  return (
    <>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="grid items-start gap-4 md:gap-6 lg:grid-cols-3"
        >
          <PageTitle
            title="新建协同 Jupyter"
            description="提供交互式的 Web 实验环境，可用于代码调试"
            className="lg:col-span-3"
            tipContent={`版本 ${MetadataFormJupyterEmias.version}`}
          >
            <div className="items-centor flex w-fit flex-row justify-end gap-3">
              <FormImportButton
                metadata={MetadataFormJupyterEmias}
                dataProcessor={dataProcessor}
                form={form}
                afterImport={(data) => {
                  if (data.envs.length > 0) {
                    setEnvOpen(true);
                  }
                  if (data.nodeSelector.enable) {
                    setOtherOpen(true);
                  }
                }}
              />
              <FormExportButton
                metadata={MetadataFormJupyterEmias}
                form={form}
              />
              <PublishConfigForm
                config={MetadataFormJupyterEmias}
                configform={form}
              />
              <LoadableButton
                isLoading={isPending}
                isLoadingText="提交作业"
                type="submit"
              >
                <CirclePlus className="size-4" />
                提交作业
              </LoadableButton>
            </div>
          </PageTitle>
          <div className="flex flex-col gap-4 md:gap-6 lg:col-span-2">
            <Card>
              <CardHeader>
                <CardTitle>基本设置</CardTitle>
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
                      <FormDescription>
                        名称可重复，最多包含 40 个字符
                      </FormDescription>
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
                />
                <ImageFormField
                  form={form}
                  name="task.image"
                  jobType={JobType.Jupyter}
                />
              </CardContent>
            </Card>
            <TemplateInfo
              form={form}
              metadata={MetadataFormJupyterEmias}
              uiStateUpdaters={[
                {
                  condition: (data) => data.envs.length > 0,
                  setter: setEnvOpen,
                  value: true,
                },
                {
                  condition: (data) =>
                    data.nodeSelector.enable || data.alertEnabled === false,
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
  );
};
