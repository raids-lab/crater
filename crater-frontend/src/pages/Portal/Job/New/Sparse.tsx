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
  apiJobTemplate,
  apiJTaskImageList,
  apiSparseCreate,
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
  volumeMountsSchema,
  envsSchema,
  taskSchema,
  convertToResourceList,
  nodeSelectorSchema,
  VolumeMountType,
  forwardsSchema,
} from "@/utils/form";
import { useEffect, useState } from "react";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { Textarea } from "@/components/ui/textarea";
import { IDlAnalyze, apiDlAnalyze } from "@/services/api/recommend/dlTask";
import { ProgressBar } from "@/components/custom/ProgressBar";
import { Cross1Icon } from "@radix-ui/react-icons";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { EnvCard } from "./Custom";
import { VolumeMountsCard } from "@/components/form/DataMountFormField";
import { OtherOptionsFormCard } from "@/components/form/OtherOptionsFormField";

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
  replicas: z.coerce.number().min(1, {
    message: "副本个数必须大于0",
  }),
  runningType: z.enum(["one-shot", "long-running"], {
    invalid_type_error: "Select a runningType",
    required_error: "请选择运行模式",
  }),
  macs: z.coerce.number(),
  params: z.coerce.number(),
  batchSize: z.coerce.number(),
  dim: z.array(
    z.object({
      vocabularySize: z.coerce.number(),
      embeddingDim: z.coerce.number(),
    }),
  ),
  task: taskSchema,
  envs: envsSchema,
  volumeMounts: volumeMountsSchema,
  alertEnabled: z.boolean().default(true),
  nodeSelector: nodeSelectorSchema,
  openssh: z.boolean().default(false),
  forwards: forwardsSchema,
});

type FormSchema = z.infer<typeof formSchema>;

export const Component = () => {
  const [envOpen, setEnvOpen] = useState<boolean>(false);
  const [otherOpen, setOtherOpen] = useState<boolean>(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAtomValue(globalUserInfo);

  const { mutate: createTask, isPending } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiSparseCreate({
        name: values.jobName,
        resource: convertToResourceList(values.task.resource),
        image: values.task.image,
        command: values.task.command,
        workingDir: values.task.workingDir,
        volumeMounts: values.volumeMounts,
        envs: values.envs,
        forwards: values.forwards,
        alertEnabled: values.alertEnabled,
        runningType: values.runningType,
        params: values.params,
        macs: values.macs,
        batchSize: values.batchSize,
        vocabularySize: values.dim.map((item) => item.vocabularySize),
        embeddingDim: values.dim.map((item) => item.embeddingDim),
        replicas: values.replicas,
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

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      jobName: "",
      runningType: "one-shot",
      macs: 0,
      params: 0,
      batchSize: 0,
      dim: [],
      task: {
        taskName: "training",
        replicas: 1,
        resource: {
          cpu: 1,
          gpu: {
            count: 0,
            model: "nvidia.com/gpu",
          },
          memory: 2,
        },
        image: "",
        command: "",
        workingDir: "/home/" + user.name,
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

      if (jobInfo.data.envs.length > 0) {
        setEnvOpen(true);
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
    fields: dimFields,
    append: dimAppend,
    remove: dimRemove,
  } = useFieldArray<FormSchema>({
    name: "dim",
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

  const [analyze, setAnalyze] = useState<IDlAnalyze>();
  const { mutate: analyzeTask } = useMutation({
    mutationFn: () =>
      apiDlAnalyze({
        runningType: form.getValues("runningType"),
        relationShips: [],
        macs: parseInt(form.getValues("macs") as unknown as string),
        params: parseInt(form.getValues("params") as unknown as string),
        batchSize: parseInt(form.getValues("batchSize") as unknown as string),
        vocabularySize: form
          .getValues("dim")
          .map((item) => parseInt(item.vocabularySize as unknown as string)),
        embeddingDim: form
          .getValues("dim")
          .map((item) => parseInt(item.embeddingDim as unknown as string)),
      }),
    onSuccess: (data) => {
      setAnalyze(data.data.data);
    },
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
              <h1 className="flex-1 shrink-0 text-xl font-semibold tracking-tight whitespace-nowrap sm:grow-0">
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

                        if (data.envs.length > 0) {
                          setEnvOpen(true);
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
              <FormField
                control={form.control}
                name="replicas"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      Pod 副本个数
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input type="number" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="runningType"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      运行模式
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Select
                        onValueChange={field.onChange}
                        defaultValue={field.value}
                      >
                        <SelectTrigger className="">
                          <SelectValue placeholder="请选择" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="one-shot">one-shot</SelectItem>
                          <SelectItem value="long-running">
                            long-running
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
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
              <div className="grid grid-cols-3 gap-3">
                <FormField
                  control={form.control}
                  name="macs"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        模型乘加运算数(百万)
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input type="number" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="params"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        模型参数量(千)
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input type="number" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="batchSize"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        BatchSize
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input type="number" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <div className="space-y-2">
                {dimFields.length > 0 && (
                  <div>
                    {dimFields.map((field, index) => (
                      <FormField
                        control={form.control}
                        key={field.id}
                        name={`dim.${index}`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel className={cn(index !== 0 && "sr-only")}>
                              稀疏特征(万)
                            </FormLabel>
                            <FormControl>
                              <div className="flex flex-row space-x-2">
                                <Input
                                  id={`input.args.${index}.vocabularySize`}
                                  type="number"
                                  placeholder="输入维度"
                                  value={field.value.vocabularySize}
                                  onChange={(event) =>
                                    field.onChange({
                                      ...field.value,
                                      vocabularySize: event.target.value,
                                    })
                                  }
                                />
                                <Input
                                  id={`input.args.${index}.embeddingDim`}
                                  type="number"
                                  placeholder="输出维度"
                                  value={field.value.embeddingDim}
                                  onChange={(event) =>
                                    field.onChange({
                                      ...field.value,
                                      embeddingDim: event.target.value,
                                    })
                                  }
                                />
                                <div>
                                  <Button
                                    size="icon"
                                    variant={"outline"}
                                    onClick={() => dimRemove(index)}
                                  >
                                    <Cross1Icon className="h-3 w-3" />
                                  </Button>
                                </div>
                              </div>
                            </FormControl>
                            {index === dimFields.length - 1 && <FormMessage />}
                          </FormItem>
                        )}
                      />
                    ))}
                  </div>
                )}
                <div className="space-x-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() =>
                      dimAppend({ embeddingDim: 0, vocabularySize: 0 })
                    }
                  >
                    添加稀疏特征维度
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => analyzeTask()}
                  >
                    分析
                  </Button>
                </div>
                {analyze && (
                  <div className="grid grid-cols-2 gap-2">
                    <Card>
                      <CardHeader className="pt-4 pb-3">
                        <CardTitle>P100 资源占用预测</CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-1.5">
                        <ProgressBar
                          width={analyze.p100.gpuUtilAvg}
                          label={`gpuUtilAvg: ${analyze.p100.gpuUtilAvg.toFixed(2)}%`}
                        />
                        <ProgressBar
                          width={(analyze.p100.gpuMemoryMaxGB / 16.0) * 100}
                          label={`gpuMemoryMaxGB: ${analyze.p100.gpuMemoryMaxGB.toFixed(
                            2,
                          )}GB`}
                        />
                      </CardContent>
                    </Card>
                    <Card>
                      <CardHeader className="pt-4 pb-3">
                        <CardTitle>V100 资源占用预测</CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-1.5">
                        <ProgressBar
                          width={analyze.v100.gpuUtilAvg}
                          label={`gpuUtilAvg: ${analyze.v100.gpuUtilAvg.toFixed(2)}%`}
                        />
                        <ProgressBar
                          width={(analyze.v100.gpuMemoryMaxGB / 32.0) * 100}
                          label={`gpuMemoryMaxGB: ${analyze.v100.gpuMemoryMaxGB.toFixed(
                            2,
                          )}GB`}
                        />
                        {/* // smActiveAvg: number;
                    // smOccupancyAvg: number;
                    // fp32ActiveAvg: number;
                    // dramActiveAvg: number; */}
                        <ProgressBar
                          width={analyze.v100.smActiveAvg}
                          label={`smActiveAvg: ${analyze.v100.smActiveAvg.toFixed(
                            2,
                          )}%`}
                        />
                        <ProgressBar
                          width={analyze.v100.smOccupancyAvg}
                          label={`smOccupancyAvg: ${analyze.v100.smOccupancyAvg.toFixed(
                            2,
                          )}%`}
                        />
                        <ProgressBar
                          width={analyze.v100.fp32ActiveAvg}
                          label={`fp32ActiveAvg: ${analyze.v100.fp32ActiveAvg.toFixed(
                            2,
                          )}%`}
                        />
                        <ProgressBar
                          width={analyze.v100.dramActiveAvg}
                          label={`dramActiveAvg: ${analyze.v100.dramActiveAvg.toFixed(
                            2,
                          )}%`}
                        />
                      </CardContent>
                    </Card>
                  </div>
                )}
              </div>
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
            <VolumeMountsCard form={form} />
            <AccordionCard
              cardTitle={EnvCard}
              open={envOpen}
              setOpen={setEnvOpen}
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
