import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import { useNavigate } from "react-router-dom";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { useForm, useFieldArray } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiJupyterCreate } from "@/services/api/vcjob";
import { convertToK8sResources } from "@/utils/resource";
import { toast } from "sonner";
import { CirclePlus, XIcon } from "lucide-react";
import FormLabelMust from "@/components/form/FormLabelMust";
import AccordionCard from "@/components/form/AccordionCard";
import { Separator } from "@/components/ui/separator";
import {
  exportToJsonString,
  nodeSelectorSchema,
  VolumeMountType,
} from "@/utils/form";
import { useState } from "react";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { IngressCard } from "./Custom";
import FormExportButton from "@/components/form/FormExportButton";
import FormImportButton from "@/components/form/FormImportButton";
import { MetadataFormJupyter } from "@/components/form/types";
import LoadableButton from "@/components/custom/LoadableButton";
import PageTitle from "@/components/layout/PageTitle";
import { PublishConfigForm } from "./Publish";
import { ImageFormField } from "@/components/form/ImageFormField";
import { VolumeMountsCard } from "@/components/form/DataMountFormField";
import { ResourceFormFields } from "@/components/form/ResourceFormField";
import { TemplateInfo } from "@/components/form/TemplateInfo";
import { OtherOptionsFormCard } from "@/components/form/OtherOptionsFormField";
import { EnvFormCard } from "@/components/form/EnvFormField";

const formSchema = z.object({
  taskname: z
    .string()
    .min(1, {
      message: "作业名称不能为空",
    })
    .max(40, {
      message: "作业名称最多包含 40 个字符",
    }),
  cpu: z.number().int().min(0, {
    message: "CPU 核数不能小于 0",
  }),
  gpu: z
    .object({
      count: z.number().int().min(0, {
        message: "指定的 GPU 卡数不能小于 0",
      }),
      model: z.string().optional(),
    })
    .refine(
      (gpu) => {
        // If a is not null, then b must not be null
        return (
          gpu.count === 0 ||
          (gpu.count > 0 && gpu.model !== null && gpu.model !== undefined)
        );
      },
      {
        message: "GPU 型号不能为空",
        path: ["model"], // The path for the error message
      },
    ),
  memory: z.number().int().min(0, {
    message: "内存大小不能小于 0",
  }),
  image: z.string().min(1, {
    message: "容器镜像不能为空",
  }),
  envs: z.array(
    z.object({
      name: z.string().min(1, {
        message: "环境变量名不能为空",
      }),
      value: z.string().min(1, {
        message: "环境变量值不能为空",
      }),
    }),
  ),
  volumeMounts: z.array(
    z.object({
      type: z.number().int(),
      subPath: z.string().min(1, {
        message: "挂载源不能为空",
      }),
      datasetID: z.number().int().nonnegative().optional(),
      mountPath: z
        .string()
        .min(1, {
          message: "挂载到容器中的路径不能为空",
        })
        .refine((value) => value.startsWith("/"), {
          message: "路径需以单个斜杠 `/` 开头",
        })
        .refine((value) => !value.includes(".."), {
          message: "禁止使用相对路径 `..`",
        })
        .refine((value) => !value.includes("//"), {
          message: "避免使用多个连续的斜杠 `//`",
        })
        .refine((value) => value !== "/", {
          message: "禁止挂载到根目录 `/`",
        }),
    }),
  ),
  ingresses: z.array(
    z.object({
      name: z
        .string()
        .min(1)
        .max(20)
        .regex(/^[a-z]+$/, {
          message: "只能包含小写字母",
        }),
      port: z.number().int().positive(),
    }),
  ),
  nodeports: z.array(
    z.object({
      name: z
        .string()
        .min(1)
        .max(20)
        .regex(/^[a-z]+$/, {
          message: "只能包含小写字母",
        }),
      port: z.number().int().positive(),
    }),
  ),
  nodeSelector: nodeSelectorSchema,
  alertEnabled: z.boolean().default(true),
});

type FormSchema = z.infer<typeof formSchema>;

export const Component = () => {
  const [envOpen, setEnvOpen] = useState(false);
  const [otherOpen, setOtherOpen] = useState(true);
  const [ingressOpen, setIngressOpen] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAtomValue(globalUserInfo);

  const { mutate: createTask, isPending } = useMutation({
    mutationFn: (values: FormSchema) => {
      const others: Record<string, string> = {};
      if (values.gpu.count > 0) {
        if (values.gpu.model) {
          others[values.gpu.model] = `${values.gpu.count}`;
        } else {
          others["nvidia.com/gpu"] = `${values.gpu.count}`;
        }
      }
      return apiJupyterCreate({
        name: values.taskname,
        resource: convertToK8sResources({
          cpu: values.cpu,
          memory: values.memory,
          others,
        }),
        image: values.image,
        volumeMounts: values.volumeMounts,
        envs: values.envs,
        ingresses: values.ingresses,
        nodeports: values.nodeports,
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
        template: exportToJsonString(MetadataFormJupyter, values),
      });
    },
    onSuccess: async (_, { taskname }) => {
      await new Promise((resolve) => setTimeout(resolve, 500)).then(() =>
        queryClient.invalidateQueries({ queryKey: ["job"] }),
      );
      toast.success(`作业 ${taskname} 创建成功`);
      navigate(-1);
    },
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      taskname: "",
      cpu: 2,
      gpu: {
        count: 0,
      },
      memory: 4,
      image: "",
      volumeMounts: [
        {
          type: VolumeMountType.FileType,
          subPath: `user`,
          mountPath: `/home/${user.name}`,
        },
      ],
      ingresses: [
        {
          name: "notebook",
          port: 8888,
        },
      ],
      envs: [],
      alertEnabled: true,
      nodeSelector: {
        enable: false,
      },
    },
  });

  const {
    fields: ingressFields,
    append: ingressAppend,
    remove: ingressRemove,
  } = useFieldArray<FormSchema>({
    name: "ingresses",
    control: form.control,
  });

  const {
    fields: nodeportFields,
    append: nodeportAppend,
    remove: nodeportRemove,
  } = useFieldArray<FormSchema>({
    name: "nodeports",
    control: form.control,
  });

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    if (isPending) {
      toast.info("请勿重复提交");
      return;
    }
    if (values.gpu.count > 0 && values.cpu <= 2 && values.memory <= 4) {
      form.setError("gpu.model", {
        type: "manual",
        message:
          "建议结合节点资源分配情况，妥善调整 CPU 和内存资源申请，避免作业被 OOM Kill",
      });
      form.setError("cpu", {
        type: "manual",
        message: "请增加 CPU 核数",
      });
      form.setError("memory", {
        type: "manual",
        message: "请增加内存大小",
      });
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
            title="新建 Jupyter Lab"
            description="提供交互式的 Web 实验环境，可用于代码调试"
            className="lg:col-span-3"
            tipContent={`版本 ${MetadataFormJupyter.version}`}
          >
            <div className="items-centor flex w-fit flex-row justify-end gap-3">
              <FormImportButton
                metadata={MetadataFormJupyter}
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
              <FormExportButton metadata={MetadataFormJupyter} form={form} />
              <PublishConfigForm
                config={MetadataFormJupyter}
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
                  name="taskname"
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
                  cpuPath="cpu"
                  memoryPath="memory"
                  gpuCountPath="gpu.count"
                  gpuModelPath="gpu.model"
                />
                <ImageFormField form={form} name="image" />
              </CardContent>
            </Card>
            <TemplateInfo
              form={form}
              metadata={MetadataFormJupyter}
              uiStateUpdaters={[
                {
                  condition: (data) => data.envs.length > 0,
                  setter: setEnvOpen,
                  value: true,
                },
                {
                  condition: (data) =>
                    data.ingresses.length > 0 || data.nodeports.length > 0,
                  setter: setIngressOpen,
                  value: true,
                },
                {
                  condition: (data) =>
                    data.nodeSelector.enable || data.alertEnabled,
                  setter: setOtherOpen,
                  value: true,
                },
              ]}
            />
          </div>
          <div className="flex flex-col gap-4 md:gap-6">
            <VolumeMountsCard form={form} />
            <AccordionCard
              cardTitle={IngressCard}
              open={ingressOpen}
              setOpen={setIngressOpen}
            >
              <div className="mt-3 space-y-5">
                <Tabs defaultValue="ingress" className="w-full">
                  <TabsList className="grid w-full grid-cols-2">
                    <TabsTrigger value="ingress">Ingress 规则</TabsTrigger>
                    <TabsTrigger value="nodeport">Nodeport 规则</TabsTrigger>
                  </TabsList>
                  <TabsContent value="ingress" className="space-y-5">
                    {ingressFields.map((field, index) => (
                      <div key={field.id}>
                        <Separator
                          className={index === 0 ? "hidden" : "mb-5"}
                        />
                        <div className="relative space-y-5">
                          <button
                            onClick={() => ingressRemove(index)}
                            className="absolute -top-1.5 right-0 rounded-sm opacity-50 transition-opacity hover:opacity-100 focus:outline-none disabled:pointer-events-none"
                          >
                            <XIcon className="size-4" />
                            <span className="sr-only">Remove</span>
                          </button>
                          <FormField
                            control={form.control}
                            name={`ingresses.${index}.name`}
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  规则名称 {index + 1}
                                  <FormLabelMust />
                                </FormLabel>
                                <FormControl>
                                  <Input {...field} />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                          <FormField
                            control={form.control}
                            name={`ingresses.${index}.port`}
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  容器端口 {index + 1}
                                  <FormLabelMust />
                                </FormLabel>
                                <FormControl>
                                  <Input
                                    {...field}
                                    type="text"
                                    onChange={(e) => {
                                      const value = e.target.value;
                                      if (value === "") {
                                        field.onChange(null);
                                      } else {
                                        const parsed = parseInt(value, 10);
                                        if (!isNaN(parsed)) {
                                          field.onChange(parsed);
                                        }
                                      }
                                    }}
                                    value={field.value ?? ""}
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </div>
                    ))}
                    <Button
                      type="button"
                      variant="secondary"
                      className="w-full"
                      onClick={() =>
                        ingressAppend({
                          name: "",
                          port: 0,
                        })
                      }
                    >
                      <CirclePlus className="size-4" />
                      添加 Ingress 规则
                    </Button>
                  </TabsContent>
                  <TabsContent value="nodeport" className="space-y-5">
                    {nodeportFields.map((field, index) => (
                      <div key={field.id}>
                        <Separator
                          className={index === 0 ? "hidden" : "mb-5"}
                        />
                        <div className="relative space-y-5">
                          <button
                            onClick={() => nodeportRemove(index)}
                            className="absolute -top-1.5 right-0 rounded-sm opacity-50 transition-opacity hover:opacity-100 focus:outline-none disabled:pointer-events-none"
                          >
                            <XIcon className="size-4" />
                            <span className="sr-only">Remove</span>
                          </button>
                          <FormField
                            control={form.control}
                            name={`nodeports.${index}.name`}
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  规则名称 {index + 1}
                                  <FormLabelMust />
                                </FormLabel>
                                <FormControl>
                                  <Input {...field} />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                          <FormField
                            control={form.control}
                            name={`nodeports.${index}.port`}
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  容器端口 {index + 1}
                                  <FormLabelMust />
                                </FormLabel>
                                <FormControl>
                                  <Input
                                    {...field}
                                    type="text"
                                    onChange={(e) => {
                                      const value = e.target.value;
                                      if (value === "") {
                                        field.onChange(null);
                                      } else {
                                        const parsed = parseInt(value, 10);
                                        if (!isNaN(parsed)) {
                                          field.onChange(parsed);
                                        }
                                      }
                                    }}
                                    value={field.value ?? ""}
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </div>
                    ))}
                    <Button
                      type="button"
                      variant="secondary"
                      className="w-full"
                      onClick={() =>
                        nodeportAppend({
                          name: "",
                          port: 0,
                        })
                      }
                    >
                      <CirclePlus className="size-4" /> 添加 Nodeport 规则
                    </Button>
                  </TabsContent>
                </Tabs>
              </div>
            </AccordionCard>
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
