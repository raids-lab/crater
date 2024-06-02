// Form in dialog: https://github.com/shadcn-ui/ui/issues/709
// Zod Number: https://github.com/shadcn-ui/ui/issues/421

import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiAiTaskCreate } from "@/services/api/aiTask";
import { cn } from "@/lib/utils";
import { Cross1Icon } from "@radix-ui/react-icons";
import { getAiKResource } from "@/utils/resource";
import { Textarea } from "@/components/ui/textarea";
// import { apiGetFiles } from "@/services/api/file";
import { toast } from "sonner";
import { FileSelectDialog } from "@/components/custom/FileSelectDialog";

const formSchema = z.object({
  taskname: z
    .string()
    .min(1, {
      message: "作业名称不能为空",
    })
    .max(40, {
      message: "作业名称最多包含40个字符",
    }),
  taskType: z.string(),
  schedulerName: z.enum(["kube-gpu-colocate-scheduler", "volcano"]),
  gpuModel: z.enum(["default", "v100", "p100", "t4", "2080ti"]),
  cpu: z.coerce.number().int().positive({ message: "CPU 核心数至少为 1" }),
  gpu: z.coerce.number().int().min(0),
  memory: z.coerce.number().int().positive(),
  // image: z.string().url(),
  image: z.string(),
  workingDir: z.string(),
  shareDirs: z.array(
    z.object({
      subPath: z.string().min(1, {
        message: "项目路径不能为空",
      }),
      mountPath: z.string(),
    }),
  ),
  command: z.string().min(1, {
    message: "作业名称不能为空",
  }),
  priority: z.enum(["low", "high"], {
    invalid_type_error: "Select a priority",
    required_error: "请选择作业优先级",
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
  const { name: currentUserName } = useAtomValue(globalUserInfo);
  // const { data: spaces } = useQuery({
  //   queryKey: ["data", "share"],
  //   queryFn: () => apiGetFiles(""),
  //   select: (res) => res.data.data,
  // });
  const convertShareDirs = (argsList: FormSchema["shareDirs"]): MountDir[] => {
    const argsDict: MountDir[] = [];
    argsList.forEach((dir) => {
      // const parts = dir.subPath.split("/");
      // const leavePath = dir.subPath.replace(`/${parts[1]}`, ""); //只会替换第一次出现

      // const top =parts[1]
      //   spaces?.find((p) => p.filename === parts[1])?.name ?? parts[1];

      // const truePath = "/" + top + leavePath;
      const truePath = dir.subPath;
      argsDict.push({
        mountPath: dir.mountPath,
        subPath: truePath,
      });
    });
    return argsDict;
  };

  const { mutate: createTask } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiAiTaskCreate({
        taskName: values.taskname,
        slo: values.priority === "high" ? 1 : 0,
        taskType: "training",
        resourceRequest: getAiKResource({
          gpu: values.gpu,
          memory: `${values.memory}Gi`,
          cpu: values.cpu,
        }),
        image: values.image,
        workingDir: values.workingDir,
        shareDirs: convertShareDirs(values.shareDirs),
        command: values.command,
        gpuModel: values.gpuModel === "default" ? "" : values.gpuModel,
        schedulerName: values.schedulerName,
      }),
    onSuccess: async (_, { taskname }) => {
      await queryClient.invalidateQueries({ queryKey: ["job", "list"] });
      toast.success(`作业 ${taskname} 创建成功`);
      closeSheet();
    },
  });

  // const shareDirsInfo = useQuery({
  //   queryKey: ["aitask", "shareDirs"],
  //   queryFn: apiAiTaskShareDirList,
  //   select: (res) => res.data.data,
  // });

  // const shareDirList = useMemo(() => {
  //   if (!shareDirsInfo.data) {
  //     return [];
  //   }
  //   return shareDirsInfo.data.map((item) => ({
  //     value: item,
  //     label: item,
  //   }));
  // }, [shareDirsInfo.data]);

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
      shareDirs: [{ subPath: "", mountPath: `/home/${currentUserName}` }],
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
    // 保存到 LocalStorage
    localStorage.setItem("training_form_cache", json);
  };

  const loadFromJson = () => {
    const json = localStorage.getItem("training_form_cache");
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
                    作业名<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  {/* <FormMessage>请输入作业名称</FormMessage> */}
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
                        <div className="grid grid-cols-2 rounded-lg border-2">
                          <div className="flex flex-row">
                            <p className="w-[5px]"></p>
                            <p className="mt-2 w-[80px] font-mono text-base">
                              项目路径:
                            </p>
                            <FileSelectDialog
                              handleSubmit={(item) =>
                                field.onChange({
                                  ...field.value,
                                  subPath: item.id,
                                })
                              }
                            />
                          </div>
                          <div className="col-span-2 flex flex-row">
                            <p className="w-[5px]"></p>
                            <p className="mt-2 w-[80px] font-mono text-base">
                              挂载路径:
                            </p>
                            <Input
                              id={`input.args.${index}.mountPath`}
                              className="w-[580px] font-mono"
                              placeholder="Mount Path"
                              value={field.value.mountPath}
                              onChange={(event) =>
                                field.onChange({
                                  ...field.value,
                                  mountPath: event.target.value,
                                })
                              }
                            />
                            <p className="w-[5px]"></p>
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
              shareDirsAppend({
                subPath: "",
                mountPath: `/home/${currentUserName}`,
              })
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
                作业优先级<span className="ml-1 text-red-500">*</span>
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
            上次作业
          </Button>
          <Button type="submit">提交作业</Button>
        </div>
      </form>
    </Form>
  );
}
