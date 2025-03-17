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
import { Input } from "@/components/ui/input";
import { useForm, useFieldArray } from "react-hook-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiTensorflowCreate } from "@/services/api/vcjob";
import { cn } from "@/lib/utils";
import { toast } from "sonner";
import {
  ChevronLeftIcon,
  CircleArrowDown,
  CircleArrowUp,
  CirclePlus,
  CirclePlusIcon,
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
import { useState } from "react";
import { Switch } from "@/components/ui/switch";
import { apiResourceList } from "@/services/api/resource";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { Textarea } from "@/components/ui/textarea";
import { ImageFormField } from "@/components/form/ImageFormField";
import { VolumeMountsCard } from "@/components/form/DataMountFormField";

const VERSION = "20240528";
const JOB_TYPE = "tensorflow";

const formSchema = z.object({
  jobName: z
    .string()
    .min(1, {
      message: "作业名称不能为空",
    })
    .max(40, {
      message: "作业名称最多包含 40 个字符",
    }),
  ps: taskSchema,
  worker: taskSchema,
  envs: envsSchema,
  volumeMounts: volumeMountsSchema,
  observability: observabilitySchema,
  alertEnabled: z.boolean().default(true),
  nodeSelector: nodeSelectorSchema,
});

type FormSchema = z.infer<typeof formSchema>;

const EnvCard = "环境变量";
const TensorboardCard = "观测面板";
const OtherCard = "其他选项";

export const Component = () => {
  const [envOpen, setEnvOpen] = useState<string>();
  const [tensorboardOpen, setTensorboardOpen] = useState<string>();
  const [otherOpen, setOtherOpen] = useState<string>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAtomValue(globalUserInfo);

  const { mutate: createTask, isPending } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiTensorflowCreate({
        name: values.jobName,
        tasks: [
          {
            name: values.ps.taskName,
            replicas: values.ps.replicas,
            resource: convertToResourceList(values.ps.resource),
            image: values.ps.image,
            command: values.ps.command,
            workingDir: values.ps.workingDir,
            ports: values.ps.ports,
          },
          {
            name: values.worker.taskName,
            replicas: values.worker.replicas,
            resource: convertToResourceList(values.worker.resource),
            image: values.worker.image,
            command: values.worker.command,
            workingDir: values.worker.workingDir,
            ports: values.worker.ports,
          },
        ],
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
      }),
    onSuccess: async (_, { jobName: taskname }) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["job"] }),
        queryClient.invalidateQueries({ queryKey: ["context", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
      toast.success(`作业 ${taskname} 创建成功`);
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

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      jobName: "",
      ps: {
        taskName: "ps",
        replicas: 1,
        resource: {
          cpu: 1,
          gpu: {
            count: 0,
          },
          memory: 2,
        },
        image: "",
        ports: [],
      },
      worker: {
        taskName: "worker",
        replicas: 1,
        resource: {
          cpu: 1,
          gpu: {
            count: 0,
          },
          memory: 2,
        },
        image: "",
        ports: [],
      },
      volumeMounts: [{ subPath: "user", mountPath: "/home/" + user.name }],
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

  const currentValues = form.watch();

  const {
    fields: envFields,
    append: envAppend,
    remove: envRemove,
  } = useFieldArray<FormSchema>({
    name: "envs",
    control: form.control,
  });

  const {
    fields: psPortFields,
    append: psPortAppend,
    remove: psPortRemove,
  } = useFieldArray<FormSchema>({
    name: "ps.ports",
    control: form.control,
  });

  const {
    fields: workerPortFields,
    append: workerPortAppend,
    remove: workerPortRemove,
  } = useFieldArray<FormSchema>({
    name: "worker.ports",
    control: form.control,
  });

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    if (isPending) {
      return;
    }
    createTask(values);
    if (isPending) {
      toast.info("请勿重复提交");
    }
  };

  return (
    <>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="grid flex-1 items-start gap-4 md:gap-x-6 lg:col-span-3 lg:grid-cols-3"
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
              <h1 className="flex-1 shrink-0 text-xl font-semibold tracking-tight whitespace-nowrap sm:grow-0">
                Tensorflow 分布式训练作业
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
                        if (data.envs.length > 0) {
                          setEnvOpen(EnvCard);
                        }
                        if (data.observability.tbEnable) {
                          setTensorboardOpen(TensorboardCard);
                        }
                        if (data.nodeSelector.enable) {
                          setOtherOpen(OtherCard);
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
          <div className="flex flex-col gap-4 lg:col-span-2">
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
                        <Input {...field} />
                      </FormControl>
                      <FormDescription>
                        名称可重复，最多包含 40 个字符
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>Parameter Server</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-5">
                <FormField
                  control={form.control}
                  name="ps.replicas"
                  render={() => (
                    <FormItem>
                      <FormLabel>
                        副本数
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          {...form.register("ps.replicas", {
                            valueAsNumber: true,
                          })}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <div className="grid grid-cols-3 gap-3">
                  <FormField
                    control={form.control}
                    name="ps.resource.cpu"
                    render={() => (
                      <FormItem>
                        <FormLabel>
                          CPU (核数)
                          <FormLabelMust />
                        </FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            {...form.register("ps.resource.cpu", {
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
                    name="ps.resource.gpu.count"
                    render={() => (
                      <FormItem>
                        <FormLabel>
                          GPU (卡数)
                          <FormLabelMust />
                        </FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            {...form.register("ps.resource.gpu.count", {
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
                    name="ps.resource.memory"
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
                              {...form.register("ps.resource.memory", {
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
                  name="ps.resource.gpu.model"
                  render={({ field }) => (
                    <FormItem hidden={currentValues.ps.resource.gpu.count == 0}>
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
                <ImageFormField form={form} name="ps.image" />
                <FormField
                  control={form.control}
                  name="ps.command"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>启动命令</FormLabel>
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
                  name="ps.workingDir"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>工作目录</FormLabel>
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
                <div className="space-y-2">
                  {psPortFields.length > 0 && (
                    <div>
                      <FormLabel>端口</FormLabel>
                    </div>
                  )}
                  {psPortFields.map((field, index) => (
                    <div key={field.id} className="flex items-start gap-2">
                      <FormField
                        control={form.control}
                        name={`ps.ports.${index}.name`}
                        render={({ field }) => (
                          <FormItem className="flex-1">
                            <FormControl>
                              <Input {...field} placeholder="端口名称" />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`ps.ports.${index}.port`}
                        render={() => (
                          <FormItem>
                            <FormControl>
                              <Input
                                type="number"
                                placeholder="端口号"
                                {...form.register(`ps.ports.${index}.port`, {
                                  valueAsNumber: true,
                                })}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <Button
                        size="icon"
                        variant="outline"
                        type="button"
                        onClick={() => psPortRemove(index)}
                      >
                        <XIcon className="size-4" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => psPortAppend({ name: "", port: 0 })}
                  >
                    <CirclePlusIcon className="size-4" />
                    添加端口
                  </Button>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>Worker</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-5">
                <FormField
                  control={form.control}
                  name="worker.replicas"
                  render={() => (
                    <FormItem>
                      <FormLabel>
                        副本数
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          {...form.register("worker.replicas", {
                            valueAsNumber: true,
                          })}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <div className="grid grid-cols-3 gap-3">
                  <FormField
                    control={form.control}
                    name="worker.resource.cpu"
                    render={() => (
                      <FormItem>
                        <FormLabel>
                          CPU (核数)
                          <FormLabelMust />
                        </FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            {...form.register("worker.resource.cpu", {
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
                    name="worker.resource.gpu.count"
                    render={() => (
                      <FormItem>
                        <FormLabel>
                          GPU (卡数)
                          <FormLabelMust />
                        </FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            {...form.register("worker.resource.gpu.count", {
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
                    name="worker.resource.memory"
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
                              {...form.register("worker.resource.memory", {
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
                  name="worker.resource.gpu.model"
                  render={({ field }) => (
                    <FormItem
                      hidden={currentValues.worker.resource.gpu.count == 0}
                    >
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
                <ImageFormField form={form} name="worker.image" />
                <FormField
                  control={form.control}
                  name="worker.command"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>启动命令</FormLabel>
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
                  name="worker.workingDir"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>工作目录</FormLabel>
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
                <div className="space-y-2">
                  {workerPortFields.length > 0 && (
                    <div>
                      <FormLabel>端口</FormLabel>
                    </div>
                  )}
                  {workerPortFields.map(({ id }, index) => (
                    <div key={id} className="flex items-start gap-2">
                      <FormField
                        control={form.control}
                        name={`worker.ports.${index}.name`}
                        render={({ field }) => (
                          <FormItem className="flex-1">
                            <FormControl>
                              <Input {...field} placeholder="端口名称" />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`worker.ports.${index}.port`}
                        render={() => (
                          <FormItem>
                            <FormControl>
                              <Input
                                type="number"
                                placeholder="端口号"
                                {...form.register(
                                  `worker.ports.${index}.port`,
                                  {
                                    valueAsNumber: true,
                                  },
                                )}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <Button
                        size="icon"
                        variant="outline"
                        onClick={() => workerPortRemove(index)}
                      >
                        <XIcon className="size-4" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => workerPortAppend({ name: "", port: 0 })}
                  >
                    <CirclePlusIcon className="size-4" />
                    添加端口
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
          <div className="flex flex-col gap-4">
            <VolumeMountsCard form={form} />
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
                              type="button"
                              onClick={() => envRemove(index)}
                              className="data-[state=open]:bg-accent data-[state=open]:text-muted-foreground absolute -top-1.5 right-0 rounded-sm opacity-50 transition-opacity hover:opacity-100 focus:outline-hidden disabled:pointer-events-none"
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
                    <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
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
                  name={`alertEnabled`}
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
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
                    <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
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
