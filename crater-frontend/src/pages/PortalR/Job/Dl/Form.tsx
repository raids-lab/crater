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
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { Cross1Icon } from "@radix-ui/react-icons";
import { getDlKResource } from "@/utils/resource";
import { apiDlTaskCreate } from "@/services/api/recommend/dlTask";

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
  datasets: z.array(z.string()),
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
        datasets: datasets.map((item) => ({ name: item })),
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
          <Button
            type="button"
            variant="outline"
            onClick={() => dimAppend({ embeddingDim: 0, vocabularySize: 0 })}
          >
            添加稀疏特征维度
          </Button>
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
