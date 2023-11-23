// Form in dialog: https://github.com/shadcn-ui/ui/issues/709
// Zod Number: https://github.com/shadcn-ui/ui/issues/421

import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm, useFieldArray } from "react-hook-form";
import { useToast } from "@/components/ui/use-toast";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { CaretSortIcon, CheckIcon, Cross1Icon } from "@radix-ui/react-icons";
import { getDlKResource } from "@/utils/resource";
import {
  IDlAnalyze,
  apiDlAnalyze,
  apiDlTaskCreate,
} from "@/services/api/recommend/dlTask";
import { useMemo, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from "@/components/ui/command";
import { apiDlDatasetList } from "@/services/api/recommend/dataset";
import { ScrollArea } from "@/components/ui/scroll-area";

const ProgressBar = ({ width, label }: { width: number; label: string }) => {
  // dramActiveAvg: 19.41%, split by :
  const [first, second] = label.split(":");
  return (
    <div className="relative h-4 rounded bg-gray-200 dark:bg-slate-700">
      <div
        className={cn("h-4 rounded", {
          "bg-green-400 dark:bg-green-700": width <= 30,
          "bg-yellow-400 dark:bg-yellow-700": width > 30 && width <= 60,
          "bg-orange-400 dark:bg-orange-700": width > 60 && width <= 80,
          "bg-red-400 dark:bg-red-700": width > 80,
        })}
        style={{ width: `${width}%` }}
      ></div>
      <div className="absolute inset-0 grid grid-cols-5 gap-1 text-xs font-medium text-foreground">
        <div className="col-span-3 text-right">{first}:</div>
        <div className="col-span-2">{second}</div>
      </div>
    </div>
  );
};

// {
//   "name": "test-recommenddljob", // 任务名称，必填
//   "replicas": 1, // pod副本个数，必填
//   "runningType": "one-shot", // 运行模式，两种取值：one-shot/long-running，必填
//   "datasets": [], // 数据集，可为空，可以选datasets/list接口里返回的数据集，列表内每个元素仅有一个name字段
//   "relationShips": [], // 暂时不管
//   "macs": 230006514, // 模型乘加运算数Macs，必填
//   "params": 66665211, // 模型参数量，必填
//   "batchSize": 64, // BatchSize， 必填
//   "vocabularySize": [5000000,10000000,600000], // 稀疏特征输入维度，前端建议和embeddingDim设计为kv形式，即列表的长度必须和embeddingDim长度一致，且元素一一对应
//   "embeddingDim": [32,32,32], // 稀疏特征输出维度
//   "template": { // pod 模板
//       "spec": {
//           "containers": [
//               {
//                   "name": "test", // 容器名称
//                   "image": "nginx:latest", // 镜像地址
//                   "resources": {
//                       "limits": {
//                           "nvidia.com/gpu": 3 // gpu个数
//                       }
//                   }
//               }
//           ]
//       }
//   }
// }
const formSchema = z.object({
  name: z
    .string()
    .min(1, {
      message: "任务名称不能为空",
    })
    .max(40, {
      message: "任务名称最多包含40个字符",
    }),
  replicas: z.coerce.number().min(1, {
    message: "副本个数必须大于0",
  }),
  runningType: z.enum(["one-shot", "long-running"], {
    invalid_type_error: "Select a runningType",
    required_error: "请选择运行模式",
  }),
  datasets: z.array(
    z.object({
      name: z.string(),
    }),
  ),
  relationShips: z.array(z.string()),
  macs: z.coerce.number(),
  params: z.coerce.number(),
  batchSize: z.coerce.number(),
  dim: z.array(
    z.object({
      vocabularySize: z.coerce.number(),
      embeddingDim: z.coerce.number(),
    }),
  ),
  containerName: z.string().min(1, { message: "容器名称不能为空" }),
  containerImage: z.string(),
  containerGpu: z.coerce.number().int().min(0),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function NewDlTaskForm({ closeSheet }: TaskFormProps) {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const datasetInfo = useQuery({
    queryKey: ["recommend", "dataset", "list"],
    queryFn: () => apiDlDatasetList(),
    select: (res) => res.data.data,
  });

  const datasetList = useMemo(() => {
    if (!datasetInfo.data) {
      return [];
    }
    const sets = datasetInfo.data.map((item) => ({
      value: item.name,
      label: item.name,
    }));
    return sets;
  }, [datasetInfo.data]);

  const { mutate: createTask } = useMutation({
    mutationFn: (values: FormSchema) => {
      const {
        containerImage,
        containerName,
        containerGpu,
        dim,
        datasets,
        ...props
      } = values;
      return apiDlTaskCreate({
        ...props,
        datasets: datasets.map((item) => item.name),
        vocabularySize: dim.map((item) => item.vocabularySize),
        embeddingDim: dim.map((item) => item.embeddingDim),
        template: {
          spec: {
            containers: [
              {
                name: containerName,
                image: containerImage,
                resources: { limits: getDlKResource({ gpu: containerGpu }) },
              },
            ],
          },
        },
      });
    },
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({ queryKey: ["dltask", "list"] });
      toast({
        title: `创建成功`,
        description: `任务 ${name} 创建成功`,
      });
      closeSheet();
    },
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      replicas: 1,
      runningType: undefined,
      datasets: [],
      relationShips: [],
      macs: 0,
      params: 0,
      batchSize: 0,
      dim: [],
      containerName: "",
      containerImage: "",
      containerGpu: 0,
    },
  });

  const {
    fields: dimFields,
    append: dimAppend,
    remove: dimRemove,
  } = useFieldArray<FormSchema>({
    name: "dim",
    control: form.control,
  });

  const {
    fields: datasetsFields,
    append: datasetsAppend,
    remove: datasetsRemove,
  } = useFieldArray<FormSchema>({
    name: "datasets",
    control: form.control,
  });

  const [analyze, setAnalyze] = useState<IDlAnalyze>();
  const { mutate: analyzeTask } = useMutation({
    mutationFn: () =>
      apiDlAnalyze({
        runningType: form.getValues("runningType"),
        relationShips: form.getValues("relationShips"),
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
    createTask(values);
  };

  return (
    <Form {...form}>
      <form
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        onSubmit={form.handleSubmit(onSubmit)}
        className="mt-6 flex flex-col space-y-4"
      >
        <div className="grid grid-cols-3 gap-3">
          <div className="col-span-2">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    任务名称<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  {/* <FormMessage>请输入任务名称</FormMessage> */}
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <FormField
            control={form.control}
            name="replicas"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Pod 副本个数<span className="ml-1 text-red-500">*</span>
                </FormLabel>
                <FormControl>
                  <Input type="number" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
        <FormField
          control={form.control}
          name="runningType"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                运行模式<span className="ml-1 text-red-500">*</span>
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
                    <SelectItem value="long-running">long-running</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="space-y-3">
          {datasetsFields.length > 0 && (
            <div>
              {datasetsFields.map((field, index) => (
                <FormField
                  control={form.control}
                  key={field.id}
                  name={`datasets.${index}`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className={cn(index !== 0 && "sr-only")}>
                        数据集
                      </FormLabel>
                      <div className="flex flex-row space-x-2">
                        <Popover modal={true}>
                          <PopoverTrigger asChild>
                            <FormControl>
                              <Button
                                variant="outline"
                                role="combobox"
                                className={cn(
                                  "w-full justify-between",
                                  !field.value && "text-muted-foreground",
                                )}
                              >
                                {field.value.name
                                  ? datasetList.find(
                                      (dataset) =>
                                        dataset.value === field.value.name,
                                    )?.label
                                  : "选择数据集"}
                                <CaretSortIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                              </Button>
                            </FormControl>
                          </PopoverTrigger>
                          <PopoverContent
                            className="p-0"
                            style={{
                              width: "var(--radix-popover-trigger-width)",
                              maxHeight:
                                "var(--radix-popover-content-available-height)",
                            }}
                          >
                            <Command>
                              <CommandInput
                                placeholder="查找数据集"
                                className="h-9"
                              />
                              <CommandEmpty>未找到匹配的数据集</CommandEmpty>
                              <ScrollArea className="h-60">
                                <CommandGroup>
                                  {datasetList.map((dataset) => (
                                    <CommandItem
                                      value={dataset.label}
                                      key={dataset.value}
                                      onSelect={() => {
                                        field.onChange({
                                          ...field.value,
                                          name: dataset.value,
                                        });
                                      }}
                                    >
                                      {dataset.label}
                                      <CheckIcon
                                        className={cn(
                                          "ml-auto h-4 w-4",
                                          dataset.value === field.value.name
                                            ? "opacity-100"
                                            : "opacity-0",
                                        )}
                                      />
                                    </CommandItem>
                                  ))}
                                </CommandGroup>
                              </ScrollArea>
                            </Command>
                          </PopoverContent>
                        </Popover>
                        <div>
                          <Button
                            size="icon"
                            variant={"outline"}
                            onClick={() => datasetsRemove(index)}
                          >
                            <Cross1Icon className="h-3 w-3" />
                          </Button>
                        </div>
                      </div>
                      {index === dimFields.length - 1 && <FormMessage />}
                    </FormItem>
                  )}
                />
              ))}
            </div>
          )}
          <Button
            type="button"
            variant="outline"
            onClick={() => datasetsAppend({ name: "" })}
          >
            添加数据集
          </Button>
        </div>
        <div className="grid grid-cols-3 gap-3">
          <FormField
            control={form.control}
            name="macs"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  模型乘加运算数<span className="ml-1 text-red-500">*</span>
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
                  模型参数量<span className="ml-1 text-red-500">*</span>
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
                  BatchSize<span className="ml-1 text-red-500">*</span>
                </FormLabel>
                <FormControl>
                  <Input type="number" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
        {/* TODO: show message, see https://stackoverflow.com/questions/76786515/how-to-display-an-error-message-with-react-hook-form-in-a-usefieldarray */}
        <div className="space-y-3">
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
                        稀疏特征
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
          <div className="space-x-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => dimAppend({ embeddingDim: 0, vocabularySize: 0 })}
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
            <div className="grid grid-cols-2 gap-3">
              <Card>
                <CardHeader className="pb-3 pt-4">
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
                <CardHeader className="pb-3 pt-4">
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
          name="containerName"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                容器名称<span className="ml-1 text-red-500">*</span>
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
          name="containerImage"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                镜像地址<span className="ml-1 text-red-500">*</span>
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
          name="containerGpu"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                GPU<span className="ml-1 text-red-500">*</span>
              </FormLabel>
              <FormControl>
                <Input type="number" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit">提交任务</Button>
      </form>
    </Form>
  );
}
