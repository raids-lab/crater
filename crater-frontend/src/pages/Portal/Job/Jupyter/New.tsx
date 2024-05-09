import { Card, CardContent } from "@/components/ui/card";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { useNavigate } from "react-router-dom";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { FileSelectDialog } from "@/components/custom/FileSelectDialog";
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
  apiJupyterCreate,
  apiJTaskImageList,
} from "@/services/api/jupyterTask";
import { cn } from "@/lib/utils";
import {
  CaretSortIcon,
  CheckIcon,
  Cross2Icon,
  PlusCircledIcon,
} from "@radix-ui/react-icons";
// import { apiGetFiles } from "@/services/api/file";
import { getAiKResource } from "@/utils/resource";
import {
  Popover,
  PopoverClose,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui-custom/popover";
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
import { ChevronLeftIcon } from "lucide-react";
import FormLabelMust from "@/components/custom/FormLabelMust";

const formSchema = z.object({
  taskname: z
    .string()
    .min(1, {
      message: "任务名称不能为空",
    })
    .max(40, {
      message: "任务名称最多包含40个字符",
    }),
  cpu: z.number().int().min(0, {
    message: "暂不支持解除 CPU 配额上限",
  }),
  gpu: z.number().int().min(0, {
    message: "指定的 GPU 卡数不能小于 0",
  }),
  memory: z.number().int().min(0, {
    message: "暂不支持解除内存配额上限",
  }),
  image: z.string().min(1, {
    message: "任务镜像不能为空",
  }),
  shareDirs: z.array(
    z.object({
      subPath: z.string().min(1, {
        message: "挂载源不能为空",
      }),
      mountPath: z.string().min(1, {
        message: "挂载到容器中的路径不能为空",
      }),
    }),
  ),
  gpuModel: z.enum(["default", "v100", "p100", "t4", "2080ti"], {
    required_error: "请选择节点 GPU 类型",
  }),
});

type FormSchema = z.infer<typeof formSchema>;

type MountDir = {
  mountPath: string;
  subPath: string;
};

const JupyterNew = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
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

      // const top = parts[1];
      // spaces?.find((p) => p.filename === parts[1])?.name ?? parts[1];

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
      apiJupyterCreate({
        name: values.taskname,
        resource: getAiKResource({
          gpu: values.gpu,
          memory: `${values.memory}Gi`,
          cpu: values.cpu,
        }),
        image: values.image,
        volumeMounts: convertShareDirs(values.shareDirs),
        nodeSelector: {},
      }),
    onSuccess: async (_, { taskname }) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["jupyter", "list"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
      toast.success(`任务 ${taskname} 创建成功`);
      navigate("/portal/job/jupyter");
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

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      taskname: "",
      cpu: 1,
      gpu: 0,
      memory: 2,
      image: "",
      shareDirs: [],
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
    <>
      <Form {...form}>
        <form
          // eslint-disable-next-line @typescript-eslint/no-misused-promises
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
                onClick={() => navigate("/portal/job/jupyter")}
              >
                <ChevronLeftIcon className="h-4 w-4" />
              </Button>
              <h1 className="flex-1 shrink-0 whitespace-nowrap text-xl font-semibold tracking-tight sm:grow-0">
                Jupyter Lab
              </h1>
            </div>
            <div className="flex flex-row gap-3">
              <Button variant={"secondary"} type="button">
                导出配置
              </Button>
              <Button
                variant={"secondary"}
                onClick={loadFromJson}
                type="button"
              >
                导入配置
              </Button>
              <Button type="submit">提交任务</Button>
            </div>
          </div>
          <Card className="lg:col-span-2">
            <CardContent className="space-y-4 p-6">
              <div className="grid grid-cols-3 gap-3">
                <div className="col-span-3">
                  <FormField
                    control={form.control}
                    name="taskname"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>
                          任务名
                          <FormLabelMust />
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
                    name="gpu"
                    render={() => (
                      <FormItem>
                        <FormLabel>
                          GPU (卡数)
                          <FormLabelMust />
                        </FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            {...form.register("gpu", { valueAsNumber: true })}
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
                          镜像地址
                          <FormLabelMust />
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
                                        (dataset) =>
                                          dataset.value === field.value,
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
                        <FormMessage />
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
                              文件挂载
                            </FormLabel>
                            <FormControl>
                              <div className="relative flex flex-col gap-2 rounded-md border border-input p-4 shadow-sm">
                                <button
                                  onClick={() => shareDirsRemove(index)}
                                  className="absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none data-[state=open]:bg-accent data-[state=open]:text-muted-foreground"
                                >
                                  <Cross2Icon className="h-4 w-4" />
                                  <span className="sr-only">Close</span>
                                </button>
                                <FileSelectDialog
                                  setSubPath={(path) =>
                                    field.onChange({
                                      ...field.value,
                                      subPath: path,
                                    })
                                  }
                                />
                                <Input
                                  id={`input.args.${index}.mountPath`}
                                  className="w-full font-mono "
                                  placeholder="Mount Path"
                                  value={field.value.mountPath}
                                  onChange={(event) =>
                                    field.onChange({
                                      ...field.value,
                                      mountPath: event.target.value,
                                    })
                                  }
                                />
                                <FormMessage />
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
                  className="w-full"
                  onClick={() =>
                    shareDirsAppend({
                      subPath: "",
                      mountPath: "",
                    })
                  }
                >
                  <PlusCircledIcon className="mr-1.5 h-4 w-4" />
                  添加需要挂载的文件
                </Button>
              </div>
              <FormField
                control={form.control}
                name="gpuModel"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      GPU 类型
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Select
                        onValueChange={field.onChange}
                        value={field.value}
                      >
                        <SelectTrigger className="">
                          <SelectValue placeholder="请选择" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="default">
                            默认（不指定）
                          </SelectItem>
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
            </CardContent>
          </Card>
          <div className="flex flex-col gap-4">
            {[
              "Parameters",
              "Secrets",
              "Environment Variables",
              "Data",
              "Tools",
            ].map((s, i) => (
              <Card key={i}>
                <Accordion type="single" collapsible>
                  <AccordionItem
                    value="item-1"
                    className="border-b-0 px-6 py-3"
                  >
                    <AccordionTrigger className="font-semibold leading-none tracking-tight hover:no-underline">
                      {s}
                    </AccordionTrigger>
                    <AccordionContent>
                      Yes. It adheres to the WAI-ARIA design pattern.
                    </AccordionContent>
                  </AccordionItem>
                </Accordion>
              </Card>
            ))}
          </div>
        </form>
      </Form>
    </>
  );
};

export default JupyterNew;
