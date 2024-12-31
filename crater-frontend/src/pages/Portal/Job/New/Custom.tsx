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
  apiJobTemplate,
  apiJTaskImageList,
  apiTrainingCreate,
  JobType,
} from "@/services/api/vcjob";
import { cn } from "@/lib/utils";
import { toast } from "sonner";
import {
  ChevronLeftIcon,
  CircleArrowDown,
  CircleArrowUp,
  CirclePlus,
  XIcon,
} from "lucide-react";
import FormLabelMust from "@/components/form/FormLabelMust";
import Combobox from "@/components/form/Combobox";
import AccordionCard from "@/components/form/AccordionCard";
import { Separator } from "@/components/ui/separator";
import {
  exportToJsonFile,
  importFromJsonFile,
  observabilitySchema,
  volumeMountsSchema,
  envsSchema,
  taskSchema,
  convertToResourceList,
  nodeSelectorSchema,
} from "@/utils/form";
import { useEffect, useState } from "react";
import { Switch } from "@/components/ui/switch";
import { apiResourceList } from "@/services/api/resource";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { Textarea } from "@/components/ui/textarea";
import { apiGetDataset } from "@/services/api/dataset";

const VERSION = "20240528";
const JOB_TYPE = "single";

const formSchema = z.object({
  jobName: z
    .string()
    .min(1, {
      message: "作业名称不能为空",
    })
    .max(40, {
      message: "作业名称最多包含 40 个字符",
    }),
  task: taskSchema,
  envs: envsSchema,
  volumeMounts: volumeMountsSchema,
  observability: observabilitySchema,
  nodeSelector: nodeSelectorSchema,
});

type FormSchema = z.infer<typeof formSchema>;

export const EnvCard = "环境变量";
export const DataMountCard = "数据挂载";
export const TensorboardCard = "观测面板";
export const OtherCard = "其他选项";

export const Component = () => {
  const [dataMountOpen, setDataMountOpen] = useState<string>();
  const [envOpen, setEnvOpen] = useState<string>();
  const [tensorboardOpen, setTensorboardOpen] = useState<string>();
  const [otherOpen, setOtherOpen] = useState<string>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAtomValue(globalUserInfo);

  const { mutate: createTask, isPending } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiTrainingCreate({
        name: values.jobName,
        resource: convertToResourceList(values.task.resource),
        image: values.task.image,
        command: values.task.command,
        workingDir: values.task.workingDir,
        volumeMounts: values.volumeMounts,
        envs: values.envs,
        useTensorBoard: values.observability.tbEnable,
        selectors: values.nodeSelector.enable
          ? [
              {
                key: "kubernetes.io/hostname",
                operator: "In",
                values: [`${values.nodeSelector.nodeName}`],
              },
            ]
          : undefined,
        template: JSON.stringify(
          {
            version: VERSION,
            type: JOB_TYPE,
            data: currentValues,
          },
          null,
          2,
        ),
      }),
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
          label: `${item.amountSingleMax}卡 · ${item.label}`,
        }));
    },
  });
  const imagesInfo = useQuery({
    queryKey: ["jupyter", "images"],
    queryFn: () => apiJTaskImageList(JobType.Custom),
    select: (res) => {
      return res.data.data.images.map((item) => ({
        value: item.imageLink,
        label: item.imageLink,
      }));
    },
  });
  const datasetInfo = useQuery({
    queryKey: ["datsets"],
    queryFn: () => apiGetDataset(),
    select: (res) => {
      return res.data.data.map((item) => ({
        value: item.url.replace(/^\/+/, ""),
        label: item.name,
      }));
    },
  });
  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      jobName: "",
      task: {
        taskName: "training",
        replicas: 1,
        resource: {
          cpu: 1,
          gpu: {
            count: 0,
          },
          memory: 2,
        },
        image: "",
        command: "",
        workingDir: "/home/" + user.name,
        ports: [],
      },
      volumeMounts: [
        { subPath: "user/" + user.space, mountPath: "/home/" + user.name },
      ],
      envs: [],
      observability: {
        tbEnable: false,
      },
      nodeSelector: {
        enable: false,
      },
    },
  });

  const { mutate: fetchJobTemplate } = useMutation({
    mutationFn: (jobName: string) => apiJobTemplate(jobName),
    onSuccess: (response) => {
      const jobInfo = JSON.parse(response.data.data);
      form.reset(jobInfo.data);
      if (jobInfo.data.volumeMounts.length > 0) {
        setDataMountOpen(DataMountCard);
      }
      if (jobInfo.data.envs.length > 0) {
        setEnvOpen(EnvCard);
      }
      if (jobInfo.data.observability.tbEnable) {
        setTensorboardOpen(TensorboardCard);
      }
      if (jobInfo.data.nodeSelector.enable) {
        setOtherOpen(OtherCard);
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
    createTask(values);
  };

  return (
    <>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="grid flex-1 items-start gap-4 md:col-span-3 md:gap-x-6 lg:grid-cols-3"
        >
          <div className="items-centor flex flex-row justify-between lg:col-span-3">
            <div className="flex flex-row items-center gap-3">
              <Button
                variant="outline"
                size="icon"
                type="button"
                className="h-8 w-8"
                onClick={() => navigate(-1)}
              >
                <ChevronLeftIcon className="size-4" />
              </Button>
              <h1 className="flex-1 shrink-0 whitespace-nowrap text-xl font-semibold tracking-tight sm:grow-0">
                单机训练作业
              </h1>
            </div>
            <div className="flex flex-row gap-3">
              <Button
                variant="outline"
                type="button"
                className="relative cursor-pointer"
              >
                <Input
                  onChange={(e) => {
                    importFromJsonFile<FormSchema>(
                      VERSION,
                      JOB_TYPE,
                      e.target.files?.[0],
                    )
                      .then((data) => {
                        form.reset(data);
                        if (data.volumeMounts.length > 0) {
                          setDataMountOpen(DataMountCard);
                        }
                        if (data.envs.length > 0) {
                          setEnvOpen(EnvCard);
                        }
                        if (data.observability.tbEnable) {
                          setTensorboardOpen(TensorboardCard);
                        }
                        toast.success(`导入配置成功`);
                      })
                      .catch(() => {
                        toast.error(`解析错误，导入配置失败`);
                      });
                  }}
                  type="file"
                  className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
                />
                <CircleArrowDown className="size-4" />
                导入配置
              </Button>
              <Button
                variant="outline"
                type="button"
                // className="h-8"
                onClick={() => {
                  form
                    .trigger()
                    .then((isValid) => {
                      if (!isValid) {
                        return;
                      }
                      exportToJsonFile(
                        {
                          version: VERSION,
                          type: JOB_TYPE,
                          data: currentValues,
                        },
                        currentValues.jobName + ".json",
                      );
                    })
                    .catch((error) => {
                      toast.error((error as Error).message);
                    });
                }}
              >
                <CircleArrowUp className="size-4" />
                导出配置
              </Button>
              <Button type="submit">
                <CirclePlus className="size-4" />
                提交作业
              </Button>
            </div>
          </div>
          <Card className="lg:col-span-2">
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
                      <Input {...field} />
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
                  name="task.resource.cpu"
                  render={() => (
                    <FormItem>
                      <FormLabel>
                        CPU (核数)
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          {...form.register("task.resource.cpu", {
                            valueAsNumber: true,
                          })}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="task.resource.gpu.count"
                  render={() => (
                    <FormItem>
                      <FormLabel>
                        GPU (卡数)
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          {...form.register("task.resource.gpu.count", {
                            valueAsNumber: true,
                          })}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="task.resource.memory"
                  render={() => (
                    <FormItem>
                      <FormLabel>
                        内存 (GB)
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <FormControl>
                          <Input
                            type="number"
                            {...form.register("task.resource.memory", {
                              valueAsNumber: true,
                            })}
                          />
                        </FormControl>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name="task.resource.gpu.model"
                render={({ field }) => (
                  <FormItem hidden={currentValues.task.resource.gpu.count == 0}>
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
              <FormField
                control={form.control}
                name="task.image"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      容器镜像
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Combobox
                        items={imagesInfo.data ?? []}
                        current={field.value}
                        handleSelect={(value) => field.onChange(value)}
                        formTitle="镜像"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="task.command"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      启动命令
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Textarea {...field} className="h-24 font-mono" />
                    </FormControl>
                    <FormDescription>
                      将覆盖镜像的启动命令，可通过{" "}
                      <span className="font-mono">;</span> 拆分多行命令
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="task.workingDir"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      工作目录
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input {...field} className="font-mono" />
                    </FormControl>
                    <FormDescription>
                      用户文件夹位于{" "}
                      <span className="font-mono">/home/{user.name}</span>
                      ，重启后数据不会丢失
                    </FormDescription>
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
                        render={({ field }) => (
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
                                  <TabsTrigger value="file">
                                    文件系统
                                  </TabsTrigger>
                                  <TabsTrigger value="dataset">
                                    数据集
                                  </TabsTrigger>
                                </TabsList>
                                <TabsContent value="file">
                                  <FileSelectDialog
                                    value={field.value.split("/").pop()}
                                    handleSubmit={(item) => {
                                      field.onChange(item.id);
                                      form.setValue(
                                        `volumeMounts.${index}.mountPath`,
                                        `/mnt/${item.realname}`,
                                      );
                                    }}
                                  />
                                </TabsContent>
                                <TabsContent value="dataset">
                                  <Combobox
                                    items={datasetInfo.data ?? []}
                                    current={field.value}
                                    handleSelect={(value) => {
                                      field.onChange(value);
                                      form.setValue(
                                        `volumeMounts.${index}.mountPath`,
                                        `/mnt/${value.split("/").pop()}`,
                                      );
                                    }}
                                    formTitle="数据集"
                                  />
                                </TabsContent>
                              </Tabs>
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`volumeMounts.${index}.mountPath`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              挂载点 {index + 1}
                              <FormLabelMust />
                            </FormLabel>
                            <FormControl>
                              <Input {...field} />
                            </FormControl>
                            <FormDescription>
                              挂载到容器中的路径
                            </FormDescription>
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
                    volumeMountAppend({
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
              cardTitle={TensorboardCard}
              value={tensorboardOpen}
              setValue={setTensorboardOpen}
            >
              <div className="mt-3 space-y-2">
                <FormField
                  control={form.control}
                  name={`observability.tbEnable`}
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between space-x-0 space-y-0">
                      <FormLabel>启用 Tensorboard</FormLabel>
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
                  name={`observability.tbLogDir`}
                  render={({ field }) => (
                    <FormItem
                      className={cn({
                        hidden: !currentValues.observability.tbEnable,
                      })}
                    >
                      <FormControl>
                        <Input {...field} className="font-mono" />
                      </FormControl>
                      <FormDescription>
                        日志路径（仅支持采集个人文件夹下日志）
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </AccordionCard>
            <AccordionCard
              cardTitle={OtherCard}
              value={otherOpen}
              setValue={setOtherOpen}
            >
              <div className="mt-3 space-y-2">
                <FormField
                  control={form.control}
                  name={`nodeSelector.enable`}
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between space-x-0 space-y-0">
                      <FormLabel>启用节点选择功能</FormLabel>
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
                        节点名称（可通过概览页面查看）
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
