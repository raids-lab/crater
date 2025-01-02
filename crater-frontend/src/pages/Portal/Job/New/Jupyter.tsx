import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import { useNavigate } from "react-router-dom";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { FileSelectDialog } from "@/components/file/FileSelectDialog";
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
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  apiJupyterCreate,
  apiJobTemplate,
  JobType,
} from "@/services/api/vcjob";
import { cn } from "@/lib/utils";
import { convertToK8sResources } from "@/utils/resource";
import { toast } from "sonner";
import { ChartNoAxesColumn, CirclePlus, XIcon } from "lucide-react";
import FormLabelMust from "@/components/form/FormLabelMust";
import Combobox, { ComboboxItem } from "@/components/form/Combobox";
import AccordionCard from "@/components/form/AccordionCard";
import { Separator } from "@/components/ui/separator";
import {
  exportToJsonString,
  importFromJsonString,
  nodeSelectorSchema,
} from "@/utils/form";
import { useEffect, useState } from "react";
import { Switch } from "@/components/ui/switch";
import { apiResourceList } from "@/services/api/resource";
import { apiGetDataset, Dataset } from "@/services/api/dataset";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { DataMountCard, EnvCard, OtherCard } from "./Custom";
import FormExportButton from "@/components/form/FormExportButton";
import FormImportButton from "@/components/form/FormImportButton";
import { MetadataFormJupyter } from "@/components/form/types";
import ImageItem from "@/components/form/ImageItem";
import useImageQuery from "@/hooks/query/useImageQuery";
import DatasetItem from "@/components/form/DatasetItem";
import { showErrorToast } from "@/utils/toast";

const FileType = 1;
const DatasetType = 2;

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
  // 添加 useTensorBoard 作为布尔类型的属性
  observability: z.object({
    tbEnable: z.boolean(),
  }),
  nodeSelector: nodeSelectorSchema,
  alertEnabled: z.boolean().default(true),
});

type FormSchema = z.infer<typeof formSchema>;

export const Component = () => {
  const [dataMountOpen, setDataMountOpen] = useState<string>(DataMountCard);
  const [envOpen, setEnvOpen] = useState<string>();
  const [otherOpen, setOtherOpen] = useState<string>();
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
        useTensorBoard: values.observability.tbEnable,
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
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["job"] }),
        queryClient.invalidateQueries({ queryKey: ["context", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
      toast.success(`作业 ${taskname} 创建成功`);
      navigate(-1);
    },
  });

  const { data: images } = useImageQuery(JobType.Jupyter);

  const datasetInfo = useQuery({
    queryKey: ["datsets"],
    queryFn: () => apiGetDataset(),
    select: (res) => {
      return res.data.data.map(
        (item) =>
          ({
            value: item.id.toString(),
            label: item.name,
            detail: item,
          }) as ComboboxItem<Dataset>,
      );
    },
  });

  const { data: resources } = useQuery({
    queryKey: ["resources", "list"],
    queryFn: () => apiResourceList(true),
    select: (res) => {
      return res.data.data
        .sort((a, b) => {
          return b.amountSingleMax - a.amountSingleMax;
        })
        .map((item) => ({
          value: item.name,
          label: `单机 ${item.amountSingleMax} 卡 · ${item.label.toUpperCase()}`,
        }));
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
          type: FileType,
          subPath: `user/${user.space}`,
          mountPath: `/home/${user.name}`,
        },
      ],
      envs: [],
      observability: {
        tbEnable: false,
      },
      alertEnabled: true,
      nodeSelector: {
        enable: false,
      },
    },
  });

  const { mutate: fetchJobTemplate } = useMutation({
    mutationFn: (jobName: string) => apiJobTemplate(jobName),
    onSuccess: (response) => {
      try {
        const jobInfo = importFromJsonString<FormSchema>(
          MetadataFormJupyter,
          response.data.data,
        );
        form.reset(jobInfo);
        if (jobInfo.volumeMounts.length > 0) {
          setDataMountOpen(DataMountCard);
        }
        if (jobInfo.envs.length > 0) {
          setEnvOpen(EnvCard);
        }
        if (jobInfo.nodeSelector.enable) {
          setOtherOpen(OtherCard);
        }
      } catch (error) {
        showErrorToast(error);
      }
    },
    onError: () => {
      toast.error("获取作业模板失败");
    },
  });

  // 检查是否有来自作业模板的参数，如果有则加载模板
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const fromJob = params.get("fromJob");
    if (fromJob) {
      fetchJobTemplate(fromJob);
    }
  }, [fetchJobTemplate]);

  const resetVolumeMountsFields = (index: number, type: number) => {
    form.setValue(`volumeMounts.${index}`, {
      subPath: "",
      type: type,
      mountPath: "",
    });
  };

  const currentValues = form.watch();

  const {
    fields: volumeMountFields,
    append: volumeMountAppend,
    remove: volumeMountRemove,
  } = useFieldArray<FormSchema>({
    name: "volumeMounts",
    control: form.control,
  });

  const {
    fields: envFields,
    append: envAppend,
    remove: envRemove,
  } = useFieldArray<FormSchema>({
    name: "envs",
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
          className="relative grid items-start gap-4 md:gap-x-6 lg:grid-cols-3"
        >
          <div className="items-centor absolute right-0 top-0 flex w-fit -translate-y-12 flex-row justify-end gap-3 lg:col-span-3">
            <FormImportButton
              metadata={MetadataFormJupyter}
              form={form}
              afterImport={(data) => {
                if (data.volumeMounts.length > 0) {
                  setDataMountOpen(DataMountCard);
                }
                if (data.envs.length > 0) {
                  setEnvOpen(EnvCard);
                }
                if (data.nodeSelector.enable) {
                  setOtherOpen(OtherCard);
                }
              }}
            />
            <FormExportButton metadata={MetadataFormJupyter} form={form} />
            <Button type="submit">
              <CirclePlus className="size-4" />
              提交作业
            </Button>
          </div>
          <Card className="lg:col-span-2">
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
              <div className="grid grid-cols-3 gap-3">
                <FormField
                  control={form.control}
                  name="cpu"
                  render={() => (
                    <FormItem>
                      <FormLabel>
                        CPU (核数)
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          {...form.register("cpu", { valueAsNumber: true })}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="memory"
                  render={() => (
                    <FormItem>
                      <FormLabel>
                        内存 (GiB)
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <FormControl>
                          <Input
                            type="number"
                            {...form.register("memory", {
                              valueAsNumber: true,
                            })}
                          />
                        </FormControl>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="gpu.count"
                  render={() => (
                    <FormItem>
                      <FormLabel>
                        GPU (卡数)
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          {...form.register("gpu.count", {
                            valueAsNumber: true,
                          })}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name="gpu.model"
                render={({ field }) => (
                  <FormItem hidden={currentValues.gpu.count == 0}>
                    <FormLabel>
                      GPU 型号
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Combobox
                        items={resources ?? []}
                        current={field.value ?? ""}
                        handleSelect={(value) => field.onChange(value)}
                        formTitle=" GPU 型号"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => toast.warning("TODO: 为用户提供资源申请分析")}
                >
                  <ChartNoAxesColumn className="size-4" />
                  资源分析
                </Button>
              </div>
              <FormField
                control={form.control}
                name={"image"}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      容器镜像
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Combobox
                        items={images ?? []}
                        current={field.value}
                        handleSelect={(value) => field.onChange(value)}
                        renderLabel={(item) => <ImageItem item={item} />}
                        formTitle="镜像"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>
          <div className="flex flex-col gap-4">
            <AccordionCard
              cardTitle={DataMountCard}
              value={dataMountOpen}
              setValue={setDataMountOpen}
            >
              <div className="mt-3 space-y-5">
                {volumeMountFields.map((field, index) => (
                  <div key={field.id}>
                    <Separator
                      className={cn("mb-5", index === 0 && "hidden")}
                    />
                    <div className="space-y-5">
                      <FormField
                        control={form.control}
                        name={`volumeMounts.${index}.subPath`}
                        render={({ field }) => {
                          const disabled =
                            form.getValues(
                              `volumeMounts.${index}.mountPath`,
                            ) === `/home/${user.name}`;
                          return (
                            <FormItem className="relative">
                              <FormLabel>
                                挂载源 {index + 1}
                                <FormLabelMust />
                              </FormLabel>
                              <button
                                onClick={() => volumeMountRemove(index)}
                                className="absolute -top-1.5 right-0 rounded-sm opacity-50 transition-opacity hover:opacity-100 focus:outline-none disabled:pointer-events-none data-[state=open]:bg-accent data-[state=open]:text-muted-foreground"
                              >
                                <XIcon className="size-4" />
                                <span className="sr-only">Close</span>
                              </button>
                              <FormControl>
                                <Tabs defaultValue="file" className="w-full">
                                  <TabsList className="grid w-full grid-cols-2">
                                    <TabsTrigger
                                      value="file"
                                      onClick={() =>
                                        resetVolumeMountsFields(index, FileType)
                                      }
                                      disabled={disabled}
                                    >
                                      文件
                                    </TabsTrigger>
                                    <TabsTrigger
                                      value="dataset"
                                      onClick={() =>
                                        resetVolumeMountsFields(
                                          index,
                                          DatasetType,
                                        )
                                      }
                                      disabled={disabled}
                                    >
                                      数据集
                                    </TabsTrigger>
                                  </TabsList>
                                  <TabsContent value="file">
                                    <FileSelectDialog
                                      value={field.value.split("/").pop()}
                                      handleSubmit={(item) => {
                                        field.onChange(item.id);
                                        form.setValue(
                                          `volumeMounts.${index}.type`,
                                          FileType,
                                        );
                                        form.setValue(
                                          `volumeMounts.${index}.mountPath`,
                                          `/data/${item.realname}`,
                                        );
                                      }}
                                      disabled={disabled}
                                    />
                                  </TabsContent>
                                  <TabsContent value="dataset">
                                    <Combobox
                                      items={datasetInfo.data ?? []}
                                      current={field.value}
                                      disabled={disabled}
                                      renderLabel={(item) => (
                                        <DatasetItem item={item} />
                                      )}
                                      handleSelect={(value) => {
                                        field.onChange(value);
                                        form.setValue(
                                          `volumeMounts.${index}.type`,
                                          DatasetType,
                                        );
                                        form.setValue(
                                          `volumeMounts.${index}.datasetID`,
                                          Number(value),
                                        );
                                        form.setValue(
                                          `volumeMounts.${index}.mountPath`,
                                          `/data/datasets-${value.split("/").pop()}`,
                                        );
                                      }}
                                      formTitle="数据集"
                                    />
                                  </TabsContent>
                                </Tabs>
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          );
                        }}
                      />
                      <FormField
                        control={form.control}
                        name={`volumeMounts.${index}.mountPath`}
                        render={({ field }) => {
                          const disabled =
                            form.getValues(
                              `volumeMounts.${index}.mountPath`,
                            ) === `/home/${user.name}`;
                          return (
                            <FormItem>
                              <FormLabel>
                                挂载点 {index + 1}
                                <FormLabelMust />
                              </FormLabel>
                              <FormControl>
                                <Input {...field} />
                              </FormControl>
                              <FormDescription>
                                {disabled
                                  ? "默认持久化用户主目录，请谨慎修改"
                                  : "可修改容器中的挂载路径"}
                              </FormDescription>
                              <FormMessage />
                            </FormItem>
                          );
                        }}
                      />
                    </div>
                  </div>
                ))}
                <Button
                  type="button"
                  variant="secondary"
                  className="w-full"
                  onClick={() =>
                    volumeMountAppend({
                      type: FileType,
                      subPath: "",
                      mountPath: "",
                    })
                  }
                >
                  <CirclePlus className="size-4" />
                  添加{DataMountCard}
                </Button>
              </div>
            </AccordionCard>{" "}
            <AccordionCard
              cardTitle={EnvCard}
              value={envOpen}
              setValue={setEnvOpen}
            >
              <div className="mt-3 space-y-5">
                {envFields.map((field, index) => (
                  <div key={field.id}>
                    <Separator
                      className={cn("mb-5", index === 0 && "hidden")}
                    />
                    <div className="space-y-5">
                      <FormField
                        control={form.control}
                        name={`envs.${index}.name`}
                        render={({ field }) => (
                          <FormItem className="relative">
                            <button
                              onClick={() => envRemove(index)}
                              className="absolute -top-1.5 right-0 rounded-sm opacity-50 transition-opacity hover:opacity-100 focus:outline-none disabled:pointer-events-none data-[state=open]:bg-accent data-[state=open]:text-muted-foreground"
                            >
                              <XIcon className="size-4" />
                              <span className="sr-only">Close</span>
                            </button>
                            <FormLabel>
                              变量名 {index + 1}
                              <FormLabelMust />
                            </FormLabel>
                            <FormControl>
                              <Input {...field} className="font-mono" />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`envs.${index}.value`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              变量值 {index + 1}
                              <FormLabelMust />
                            </FormLabel>
                            <FormControl>
                              <Input {...field} className="font-mono" />
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
                    envAppend({
                      name: "",
                      value: "",
                    })
                  }
                >
                  <CirclePlus className="size-4" />
                  添加{EnvCard}
                </Button>
              </div>
            </AccordionCard>
            <AccordionCard
              cardTitle={OtherCard}
              value={otherOpen}
              setValue={setOtherOpen}
            >
              <div className="mt-3 space-y-3">
                <FormField
                  control={form.control}
                  name={`alertEnabled`}
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between space-x-0 space-y-0">
                      <FormLabel className="font-normal">
                        接收状态通知
                      </FormLabel>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`nodeSelector.enable`}
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between space-x-0 space-y-0">
                      <FormLabel className="font-normal">
                        指定工作节点
                      </FormLabel>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`nodeSelector.nodeName`}
                  render={({ field }) => (
                    <FormItem
                      className={cn({
                        hidden: !currentValues.nodeSelector.enable,
                      })}
                    >
                      <FormControl>
                        <Input {...field} className="font-mono" />
                      </FormControl>
                      <FormDescription>
                        输入节点名称（可通过概览页面查看）
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </AccordionCard>
          </div>
        </form>
      </Form>
    </>
  );
};
