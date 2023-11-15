// Form in dialog: https://github.com/shadcn-ui/ui/issues/709
// Zod Number: https://github.com/shadcn-ui/ui/issues/421

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
import { useToast } from "@/components/ui/use-toast";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiAiTaskCreate } from "@/services/api/aiTask";
import { showErrorToast } from "@/utils/toast";
import { cn } from "@/lib/utils";
import { Cross1Icon } from "@radix-ui/react-icons";
import { logger } from "@/utils/loglevel";
import { getKubernetesResource } from "@/utils/resource";

const formSchema = z.object({
  taskname: z
    .string()
    .min(1, {
      message: "任务名称不能为空",
    })
    .max(40, {
      message: "任务名称最多包含40个字符",
    }),
  cpu: z.number().int().positive({ message: "CPU 核心数至少为 1" }),
  gpu: z.number().int().min(0),
  memory: z.number().int().positive(),
  // image: z.string().url(),
  image: z.string(),
  workingDir: z.string(),
  shareDirs: z.array(
    z.object({
      key: z.string(),
      value: z.string(),
    }),
  ),
  command: z.string().min(1, {
    message: "任务名称不能为空",
  }),
  args: z.array(
    z.object({
      key: z.string(),
      value: z.string(),
    }),
  ),
  priority: z.enum(["low", "high"], {
    invalid_type_error: "Select a priority",
    required_error: "请选择任务优先级",
  }),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function NewTaskForm({ closeSheet }: TaskFormProps) {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const convertArgs = (
    argsList: { key: string; value: string }[],
  ): { [key: string]: string } => {
    const argsDict: { [key: string]: string } = {};
    argsList
      .filter((arg) => arg.key.length > 0)
      .forEach((arg) => {
        argsDict[arg.key] = arg.value ?? "";
      });
    return argsDict;
  };

  const { mutate: createTask } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiAiTaskCreate({
        taskName: values.taskname,
        slo: values.priority === "high" ? 1 : 0,
        taskType: "training",
        resourceRequest: getKubernetesResource({
          gpu: values.gpu,
          memory: `${values.memory}Gi`,
          cpu: values.cpu,
        }),
        image: values.image,
        workingDir: values.workingDir,
        shareDirs: convertArgs(values.shareDirs),
        command: values.command,
        args: convertArgs(values.args),
      }),
    onSuccess: (_, { taskname }) => {
      queryClient
        .invalidateQueries({ queryKey: ["aitask", "list"] })
        .then(() => {
          toast({
            title: `创建成功`,
            description: `任务 ${taskname} 创建成功`,
          });
          closeSheet();
        })
        .catch((err) => {
          showErrorToast("刷新任务列表失败", err);
        });
    },
    onError: (err) => showErrorToast("创建失败", err),
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      taskname: "",
      cpu: 1,
      gpu: 0,
      memory: 8,
      image: "",
      workingDir: "",
      shareDirs: [{ key: "", value: "" }],
      command: "",
      args: [{ key: "", value: "" }],
      priority: undefined,
    },
  });

  const {
    fields: argsFields,
    append: argsAppend,
    remove: argsRemove,
  } = useFieldArray<FormSchema>({
    name: "args",
    control: form.control,
  });

  const {
    fields: shareDirsFields,
    append: shareDirsAppend,
    remove: shareDirsRemove,
  } = useFieldArray<FormSchema>({
    name: "shareDirs",
    control: form.control,
  });

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    toast({ title: values.taskname });
    logger.debug(convertArgs(values.args));
    createTask(values);
  };

  return (
    <Form {...form}>
      <form
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        onSubmit={form.handleSubmit(onSubmit)}
        className="mt-6 flex flex-col space-y-4"
      >
        <FormField
          control={form.control}
          name="taskname"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                任务名<span className="ml-1 text-red-500">*</span>
              </FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              {/* <FormMessage>请输入任务名称</FormMessage> */}
              <FormMessage />
            </FormItem>
          )}
        />
        <div>
          <div className="grid grid-cols-3 gap-3">
            <FormField
              control={form.control}
              name="cpu"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    CPU<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      {...field}
                      onChange={(event) => field.onChange(+event.target.value)}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="gpu"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    GPU<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      {...field}
                      onChange={(event) => field.onChange(+event.target.value)}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="memory"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    内存 (GB)<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      {...field}
                      onChange={(event) => field.onChange(+event.target.value)}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          {/* <FormDescription className="mt-2">
            请选择需要分配的机器配置
          </FormDescription> */}
        </div>
        <FormField
          control={form.control}
          name="image"
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
          name="workingDir"
          render={({ field }) => (
            <FormItem>
              <FormLabel>执行目录</FormLabel>
              <FormControl>
                <Input {...field} className="font-mono" />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="space-y-3">
          {argsFields.length > 0 && (
            <div>
              {shareDirsFields.map((field, index) => (
                <FormField
                  control={form.control}
                  key={field.id}
                  name={`shareDirs.${index}`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className={cn(index !== 0 && "sr-only")}>
                        共享目录
                      </FormLabel>
                      <FormControl>
                        <div className="flex flex-row space-x-2">
                          <Input
                            id={`input.args.${index}.key`}
                            className="font-mono"
                            value={field.value.key}
                            onChange={(event) =>
                              field.onChange({
                                ...field.value,
                                key: event.target.value,
                              })
                            }
                          />
                          <Input
                            id={`input.args.${index}.value`}
                            className="font-mono"
                            value={field.value.value}
                            onChange={(event) =>
                              field.onChange({
                                ...field.value,
                                value: event.target.value,
                              })
                            }
                          />
                          <div>
                            <Button
                              size="icon"
                              variant="secondary"
                              onClick={() => shareDirsRemove(index)}
                            >
                              <Cross1Icon className="h-3 w-3" />
                            </Button>
                          </div>
                        </div>
                      </FormControl>
                    </FormItem>
                  )}
                />
              ))}
            </div>
          )}
          <Button
            type="button"
            variant="outline"
            onClick={() => shareDirsAppend({ key: "", value: "" })}
          >
            添加共享目录
          </Button>
        </div>
        <FormField
          control={form.control}
          name="command"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                执行命令<span className="ml-1 text-red-500">*</span>
              </FormLabel>
              <FormControl>
                <Input {...field} className="font-mono" />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="space-y-3">
          {argsFields.length > 0 && (
            <div>
              {argsFields.map((field, index) => (
                <FormField
                  control={form.control}
                  key={field.id}
                  name={`args.${index}`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className={cn(index !== 0 && "sr-only")}>
                        命令参数
                      </FormLabel>
                      <FormDescription className={cn(index !== 0 && "sr-only")}>
                        请在左侧输入 Key，右侧输入 Value
                      </FormDescription>
                      <FormControl>
                        <div className="flex flex-row space-x-2">
                          <Input
                            id={`input.args.${index}.key`}
                            className="font-mono"
                            value={field.value.key}
                            onChange={(event) =>
                              field.onChange({
                                ...field.value,
                                key: event.target.value,
                              })
                            }
                          />
                          <Input
                            id={`input.args.${index}.value`}
                            className="font-mono"
                            value={field.value.value}
                            onChange={(event) =>
                              field.onChange({
                                ...field.value,
                                value: event.target.value,
                              })
                            }
                          />
                          <div>
                            <Button
                              size="icon"
                              variant="secondary"
                              onClick={() => argsRemove(index)}
                            >
                              <Cross1Icon className="h-3 w-3" />
                            </Button>
                          </div>
                        </div>
                      </FormControl>
                    </FormItem>
                  )}
                />
              ))}
            </div>
          )}
          <Button
            type="button"
            variant="outline"
            onClick={() => argsAppend({ key: "", value: "" })}
          >
            添加命令参数
          </Button>
        </div>
        <FormField
          control={form.control}
          name="priority"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                任务优先级<span className="ml-1 text-red-500">*</span>
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
                    <SelectItem value="high">高优先级</SelectItem>
                    <SelectItem value="low">低优先级</SelectItem>
                  </SelectContent>
                </Select>
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
