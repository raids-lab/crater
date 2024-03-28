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

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  apiJTaskCreate,
  apiJTaskShareDirList,
  apiJTaskImageList,
} from "@/services/api/jupyterTask";
import { cn } from "@/lib/utils";
import { CaretSortIcon, CheckIcon, Cross1Icon } from "@radix-ui/react-icons";
import { getAiKResource } from "@/utils/resource";
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
import { toast } from "sonner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const formSchema = z.object({
  taskname: z
    .string()
    .min(1, {
      message: "任务名称不能为空",
    })
    .max(40, {
      message: "任务名称最多包含40个字符",
    }),
  //taskType: z.string(),
  cpu: z.coerce.number().int().min(-1),
  gpu: z.coerce.number().int().min(-1),
  memory: z.coerce.number().int().min(-1),
  schedulerName: z.enum(["kube-gpu-colocate-scheduler", "volcano"]),
  gpuModel: z.enum(["default", "v100", "p100", "t4", "2080ti"]),
  // image: z.string().url(),
  image: z.string().min(1, {
    message: "任务镜像不能为空",
  }),
  //workingDir: z.string(),
  shareDirs: z.array(
    z.object({
      path: z.string().min(1, {
        message: "共享目录不能为空",
      }),
      subPath: z.string(),
      mountPath: z.string(),
    }),
  ),
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
      apiJTaskCreate({
        taskName: values.taskname,
        taskType: "jupyter",
        resourceRequest: getAiKResource({
          gpu: values.gpu,
          memory: `${values.memory}Gi`,
          cpu: values.cpu,
        }),
        image: values.image,
        //workingDir: values.workingDir,
        shareDirs: convertShareDirs(values.shareDirs),
        //command: values.command,
        gpuModel: values.gpuModel === "default" ? "" : values.gpuModel,
        schedulerName: values.schedulerName,
      }),
    onSuccess: async (_, { taskname }) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["jupyter", "list"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
      toast.success(`任务 ${taskname} 创建成功`);
      closeSheet();
    },
  });

  const imagesInfo = useQuery({
    queryKey: ["jupyter", "images"],
    queryFn: apiJTaskImageList,
    select: (res) => res.data.data.images,
  });

  const imageList = useMemo(() => {
    if (!imagesInfo.data) {
      return [];
    }
    return imagesInfo.data.map((item) => ({
      value: item,
      label: item,
    }));
  }, [imagesInfo.data]);

  const shareDirsInfo = useQuery({
    queryKey: ["jupyter", "shareDirs"],
    queryFn: apiJTaskShareDirList,
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
      //taskType: "",
      cpu: 1,
      gpu: 0,
      memory: 8,
      image: "",
      //workingDir: "",
      shareDirs: [{ subPath: "", mountPath: "" }],
      //command: "",
      //priority: undefined,
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

  // const {
  //   fields: imagesFields,
  //   append: imagesAppend,
  //   remove: imagesRemove,
  // } = useFieldArray<FormSchema>({
  //   name: "image",
  //   control: form.control,
  // });

  const exportToJson = () => {
    const json = JSON.stringify(currentValues, null, 2);
    // 保存到 LocalStorage
    localStorage.setItem("jupyter_form_cache", json);
  };

  const loadFromJson = () => {
    const json = localStorage.getItem("jupyter_form_cache");
    if (json) {
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
      const data: FormSchema = JSON.parse(json);
      form.reset(data);
    }
  };

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    exportToJson();
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
          <div className="col-span-3">
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
        <div className="space-y-2">
          <div>
            <FormField
              control={form.control}
              name={`image`}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像地址<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    {/* <div className="flex flex-row space-x-2"> */}
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
                            {field.value
                              ? imageList.find(
                                  (dataset) => dataset.value === field.value,
                                )?.label
                              : "选择镜像"}
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
                            placeholder="查找镜像"
                            className="h-9"
                          />
                          <CommandEmpty>未找到匹配的镜像</CommandEmpty>
                          <CommandGroup>
                            {imageList.map((image) => (
                              <CommandItem
                                value={image.label}
                                key={image.value}
                                onSelect={() => {
                                  field.onChange(image.value);
                                }}
                              >
                                <PopoverClose
                                  key={image.value}
                                  className="flex w-full flex-row items-center justify-between font-mono"
                                >
                                  {image.label}
                                  <CheckIcon
                                    className={cn(
                                      "ml-auto h-4 w-4",
                                      image.value === field.value
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
                  </FormControl>
                </FormItem>
              )}
            />
          </div>
        </div>
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
          name="schedulerName"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                调度器<span className="ml-1 text-red-500">*</span>
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
                    <SelectItem value="kube-gpu-colocate-scheduler">
                      ACT Lucid
                    </SelectItem>
                    <SelectItem value="volcano">Huawei Volcano</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="gpuModel"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                GPU 类型<span className="ml-1 text-red-500">*</span>
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
                    <SelectItem value="default">默认（不指定）</SelectItem>
                    <SelectItem value="v100">
                      NVIDIA-Tesla V100-SXM2-32GB
                    </SelectItem>
                    <SelectItem value="p100">
                      NVIDIA-Tesla P100-PCIE-16GB
                    </SelectItem>
                    <SelectItem value="t4">NVIDIA-Tesla T4</SelectItem>
                    <SelectItem value="2080ti">
                      NVIDIA GeForce RTX 2080 Ti
                    </SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="grid grid-cols-2 gap-3">
          <Button variant={"secondary"} onClick={loadFromJson} type="button">
            上次任务
          </Button>
          <Button type="submit">提交 Jupyter 任务</Button>
        </div>
      </form>
    </Form>
  );
}
