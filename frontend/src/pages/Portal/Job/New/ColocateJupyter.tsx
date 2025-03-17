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
import {
  apiJupyterCreate,
  apiJTaskImageList,
  JobType,
  apiJobTemplate,
} from "@/services/api/vcjob";
import { cn } from "@/lib/utils";
import { convertToK8sResources } from "@/utils/resource";
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
  nodeSelectorSchema,
  volumeMountsSchema,
  VolumeMountType,
} from "@/utils/form";
import { useEffect, useState } from "react";
import { Switch } from "@/components/ui/switch";
import { apiResourceList } from "@/services/api/resource";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { EnvCard, TensorboardCard, OtherCard } from "./Custom";
import { VolumeMountsCard } from "@/components/form/DataMountFormField";

const VERSION = "20240528";
const JOB_TYPE = "jupyter";

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
  volumeMounts: volumeMountsSchema,
  // 添加 useTensorBoard 作为布尔类型的属性
  observability: z
    .object({
      tbEnable: z.boolean(),
      tbLogDir: z.string().optional(),
    })
    .refine(
      (observability) => {
        return (
          !observability.tbEnable ||
          (observability.tbEnable &&
            observability.tbLogDir !== null &&
            observability.tbLogDir !== undefined)
        );
      },
      {
        message: "TensorBoard 日志目录不能为空",
        path: ["tbLogDir"],
      },
    ),
  alertEnabled: z.boolean().default(true),
  nodeSelector: nodeSelectorSchema,
  openssh: z.boolean().default(false),
});

type FormSchema = z.infer<typeof formSchema>;

export const Component = () => {
  const [envOpen, setEnvOpen] = useState<string>();
  const [tensorboardOpen, setTensorboardOpen] = useState<string>();
  const [otherOpen, setOtherOpen] = useState<string>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAtomValue(globalUserInfo);

  const { mutate: createTask, isPending } = useMutation({
    mutationFn: (values: FormSchema) => {
      const others: Record<string, string> = {};
      if (values.gpu.model) {
        others[values.gpu.model] = `${values.gpu.count}`;
      } else {
        others["nvidia.com/gpu"] = `${values.gpu.count}`;
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
        alertEnabled: values.alertEnabled,
        openssh: values.openssh,
        ingresses: values.ingresses,
        nodeports: values.nodeports,
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

  const imagesInfo = useQuery({
    queryKey: ["jupyter", "images"],
    queryFn: () => apiJTaskImageList(JobType.Jupyter),
    select: (res) => {
      return res.data.data.images.map((item) => ({
        value: item.imageLink,
        label: item.imageLink,
      }));
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
      taskname: "",
      cpu: 1,
      gpu: {
        count: 0,
      },
      memory: 2,
      image: "",
      volumeMounts: [
        {
          type: VolumeMountType.FileType,
          subPath: `user`,
          mountPath: `/home/${user.name}`,
        },
      ],
      envs: [],
      observability: {
        tbEnable: false,
      },
      nodeSelector: {
        enable: false,
      },
      ingresses: [{ name: "notebook", port: 8888 }],
      nodeports: [],
    },
  });

  const { mutate: fetchJobTemplate } = useMutation({
    mutationFn: (jobName: string) => apiJobTemplate(jobName),
    onSuccess: (response) => {
      const jobInfo = JSON.parse(response.data.data);
      form.reset(jobInfo.data);

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
          className="grid items-start gap-4 md:col-span-3 md:gap-x-6 lg:grid-cols-3"
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
              <h1 className="shrink-0 text-xl font-semibold tracking-tight whitespace-nowrap sm:grow-0">
                Jupyter Lab
              </h1>
            </div>
            <div className="flex flex-row gap-3">
              <Button
                variant="outline"
                type="button"
                className="relative h-8 cursor-pointer"
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
                className="h-8"
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
                        currentValues.taskname + ".json",
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
              <Button type="submit" className="h-8">
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
                name="taskname"
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
                <FormField
                  control={form.control}
                  name="memory"
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
              <FormField
                control={form.control}
                name={`image`}
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
            </CardContent>
          </Card>
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
