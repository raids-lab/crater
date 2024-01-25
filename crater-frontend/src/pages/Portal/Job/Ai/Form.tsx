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
import { apiAiTaskCreate, apiAiTaskShareDirList } from "@/services/api/aiTask";
import { cn } from "@/lib/utils";
import { CaretSortIcon, CheckIcon, Cross1Icon } from "@radix-ui/react-icons";
import { getAiKResource } from "@/utils/resource";
import { Textarea } from "@/components/ui/textarea";
import {
  Popover,
  PopoverClose,
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
import { useMemo } from "react";

const formSchema = z.object({
  taskname: z
    .string()
    .min(1, {
      message: "任务名称不能为空",
    })
    .max(40, {
      message: "任务名称最多包含40个字符",
    }),
  taskType: z.string(),
  cpu: z.coerce.number().int().positive({ message: "CPU 核心数至少为 1" }),
  gpu: z.coerce.number().int().min(0),
  memory: z.coerce.number().int().positive(),
  // image: z.string().url(),
  image: z.string(),
  workingDir: z.string(),
  shareDirs: z.array(
    z.object({
      path: z.string(),
      subPath: z.string(),
      mountPath: z.string(),
    }),
  ),
  command: z.string().min(1, {
    message: "任务名称不能为空",
  }),
  priority: z.enum(["low", "high"], {
    invalid_type_error: "Select a priority",
    required_error: "请选择任务优先级",
  }),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

type MountDir = {
  mountPath: string;
  subPath?: string;
};

export function NewTaskForm({ closeSheet }: TaskFormProps) {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const convertShareDirs = (
    argsList: FormSchema["shareDirs"],
  ): { [key: string]: MountDir[] } => {
    const argsDict: { [key: string]: MountDir[] } = {};
    argsList
      .filter((dir) => dir.path.length > 0)
      .forEach((dir) => {
        if (!argsDict[dir.path]) {
          argsDict[dir.path] = [];
        }
        argsDict[dir.path].push(
          dir.subPath
            ? {
                mountPath: dir.mountPath,
                subPath: dir.subPath,
              }
            : {
                mountPath: dir.mountPath,
              },
        );
      });
    return argsDict;
  };

  const { mutate: createTask } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiAiTaskCreate({
        taskName: values.taskname,
        slo: values.priority === "high" ? 1 : 0,
        taskType: values.taskType,
        resourceRequest: getAiKResource({
          gpu: values.gpu,
          memory: `${values.memory}Gi`,
          cpu: values.cpu,
        }),
        image: values.image,
        workingDir: values.workingDir,
        shareDirs: convertShareDirs(values.shareDirs),
        command: values.command,
      }),
    onSuccess: async (_, { taskname }) => {
      await queryClient.invalidateQueries({ queryKey: ["aitask", "list"] });
      toast({
        title: `创建成功`,
        description: `任务 ${taskname} 创建成功`,
      });
      closeSheet();
    },
  });

  const shareDirsInfo = useQuery({
    queryKey: ["aitask", "shareDirs"],
    queryFn: apiAiTaskShareDirList,
    select: (res) => res.data.data,
  });

  const shareDirList = useMemo(() => {
    if (!shareDirsInfo.data) {
      return [];
    }
    return shareDirsInfo.data.map((item) => ({
      value: item,
      label: item,
    }));
  }, [shareDirsInfo.data]);

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      taskname: "",
      taskType: "",
      cpu: 1,
      gpu: 0,
      memory: 8,
      image: "",
      workingDir: "",
      shareDirs: [{ subPath: "", mountPath: "" }],
      command: "",
      priority: undefined,
    },
  });

  const currentValues = form.watch();

  const {
    fields: shareDirsFields,
    append: shareDirsAppend,
    remove: shareDirsRemove,
  } = useFieldArray<FormSchema>({
    name: "shareDirs",
    control: form.control,
  });

  const exportToJson = () => {
    const json = JSON.stringify(currentValues, null, 2);
    console.log(json); // 或者可以保存到文件
    // 复制到剪贴板
    navigator.clipboard.writeText(json).then(
      () => {
        toast({ title: "复制成功" });
      },
      () => {
        toast({ title: "复制失败" });
      },
    );
  };

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    toast({ title: values.taskname });
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
          </div>
          <FormField
            control={form.control}
            name="taskType"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  任务类型<span className="ml-1 text-red-500">*</span>
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
                    <Input type="number" {...field} />
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
                    <Input type="number" {...field} />
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
                    <Input type="number" {...field} />
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
        <div className="space-y-2">
          {shareDirsFields.length > 0 && (
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
                          <Popover modal={true}>
                            <PopoverTrigger asChild>
                              <FormControl>
                                <Button
                                  variant="outline"
                                  role="combobox"
                                  className={cn(
                                    "w-full justify-between text-ellipsis whitespace-nowrap",
                                    !field.value && "text-muted-foreground",
                                  )}
                                >
                                  {field.value.path
                                    ? shareDirList.find(
                                        (dataset) =>
                                          dataset.value === field.value.path,
                                      )?.label
                                    : "选择目录"}
                                  <CaretSortIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                                </Button>
                              </FormControl>
                            </PopoverTrigger>
                            <PopoverContent
                              className="p-0"
                              style={{
                                maxHeight:
                                  "var(--radix-popover-content-available-height)",
                              }}
                            >
                              <Command>
                                <CommandInput
                                  placeholder="查找目录"
                                  className="h-9"
                                />
                                <CommandEmpty>未找到匹配的数据集</CommandEmpty>
                                <CommandGroup>
                                  {shareDirList.map((shareDir) => (
                                    <CommandItem
                                      value={shareDir.label}
                                      key={shareDir.value}
                                      onSelect={() => {
                                        field.onChange({
                                          ...field.value,
                                          path: shareDir.value,
                                        });
                                      }}
                                    >
                                      <PopoverClose
                                        key={shareDir.value}
                                        className="flex w-full flex-row items-center justify-between font-mono"
                                      >
                                        {shareDir.label}
                                        <CheckIcon
                                          className={cn(
                                            "ml-auto h-4 w-4",
                                            shareDir.value === field.value.path
                                              ? "opacity-100"
                                              : "opacity-0",
                                          )}
                                        />
                                      </PopoverClose>
                                    </CommandItem>
                                  ))}
                                </CommandGroup>
                              </Command>
                            </PopoverContent>
                          </Popover>
                          <Input
                            id={`input.args.${index}.subPath`}
                            className="font-mono"
                            placeholder="Sub Path"
                            value={field.value.subPath}
                            onChange={(event) =>
                              field.onChange({
                                ...field.value,
                                subPath: event.target.value,
                              })
                            }
                          />
                          <Input
                            id={`input.args.${index}.mountPath`}
                            className="font-mono"
                            placeholder="Mount Path"
                            value={field.value.mountPath}
                            onChange={(event) =>
                              field.onChange({
                                ...field.value,
                                mountPath: event.target.value,
                              })
                            }
                          />
                          <div>
                            <Button
                              size="icon"
                              variant="outline"
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
            onClick={() =>
              shareDirsAppend({ path: "", subPath: "", mountPath: "" })
            }
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
                <Textarea {...field} className="font-mono" />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
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
        <div className="grid grid-cols-4 gap-3">
          <Button variant={"secondary"} onClick={exportToJson} type="button">
            复制任务
          </Button>
          <Button variant={"secondary"}>导入任务</Button>
          <Button type="submit" className="col-span-2">
            提交任务
          </Button>
        </div>
      </form>
    </Form>
  );
}
