import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import { useNavigate } from "react-router-dom";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { FileSelectDialog } from "@/components/custom/FileSelectDialog";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { useForm, useFieldArray } from "react-hook-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  apiJupyterCreate,
  apiJTaskImageList,
} from "@/services/api/jupyterTask";
import { cn } from "@/lib/utils";
import { Cross2Icon } from "@radix-ui/react-icons";
import { getAiKResource } from "@/utils/resource";
import { toast } from "sonner";
import {
  ChevronLeftIcon,
  CircleArrowDown,
  CircleArrowUp,
  CirclePlus,
} from "lucide-react";
import FormLabelMust from "@/components/custom/FormLabelMust";
import Combobox from "@/components/form/Combobox";
import { apiNodeLabelsList } from "@/services/api/nodelabel";
import AccordionCard from "@/components/custom/AccordionCard";
import { Separator } from "@/components/ui/separator";
import { exportToJson, importFromJson } from "@/utils/form";
import { useState } from "react";

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
    message: "暂不支持解除 CPU 配额上限",
  }),
  gpu: z.number().int().min(0, {
    message: "指定的 GPU 卡数不能小于 0",
  }),
  memory: z.number().int().min(0, {
    message: "暂不支持解除内存配额上限",
  }),
  gpuModel: z.string().min(1, {
    message: "请选择节点 GPU 类型",
  }),
  image: z.string().min(1, {
    message: "作业镜像不能为空",
  }),
  shareDirs: z.array(
    z.object({
      subPath: z.string().min(1, {
        message: "挂载源不能为空",
      }),
      mountPath: z
        .string()
        .min(1, {
          message: "挂载到容器中的路径不能为空",
        })
        .refine((value) => value.startsWith("/"), {
          message: "路径需以单个斜杠 `/` 开头",
        })
        .refine((value) => !value.includes(".."), {
          message: "禁止使用相对路径 `..`",
        })
        .refine((value) => !value.includes("//"), {
          message: "避免使用多个连续的斜杠 `//`",
        })
        .refine((value) => value !== "/", {
          message: "禁止挂载到根目录 `/`",
        }),
    }),
  ),
});

type FormSchema = z.infer<typeof formSchema>;

type MountDir = {
  mountPath: string;
  subPath: string;
};

const DataMount = "数据挂载";

const JupyterNew = () => {
  const [dataMountOpen, setDataMountOpen] = useState<string>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const convertShareDirs = (argsList: FormSchema["shareDirs"]): MountDir[] => {
    const argsDict: MountDir[] = [];
    argsList.forEach((dir) => {
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
        products:
          values.gpuModel !== "default"
            ? labelsInfo.data?.dict.get(values.gpuModel) ?? []
            : [],
      }),
    onSuccess: async (_, { taskname }) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["jupyter", "list"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
      toast.success(`作业 ${taskname} 创建成功`);
      navigate("/portal/job/jupyter");
    },
  });

  const imagesInfo = useQuery({
    queryKey: ["jupyter", "images"],
    queryFn: apiJTaskImageList,
    select: (res) => {
      return res.data.data.images.map((item) => ({
        value: item,
        label: item,
      }));
    },
  });

  const labelsInfo = useQuery({
    queryKey: ["label", "list"],
    queryFn: apiNodeLabelsList,
    select: (res) => {
      const items = res.data.data.sort((a, b) => {
        // make name is empty string to be the first
        if (a.name === "") {
          return -1;
        }
        if (b.name === "") {
          return 1;
        }
        return b.priority - a.priority;
      });
      // create a map, key is item.name, value is [item.label]
      const dict = new Map<string, string[]>();
      const list = new Array<{
        value: string;
        label: string;
      }>();
      items.forEach((item) => {
        if (dict.has(item.name)) {
          dict.get(item.name)?.push(item.label);
        } else {
          dict.set(item.name, [item.label]);
          list.push({
            value: item.name === "" ? "default" : item.name,
            label: item.name === "" ? "默认 (不指定类型)" : item.name,
          });
        }
      });
      return {
        dict,
        list,
      };
    },
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      taskname: "",
      cpu: 1,
      gpu: 0,
      memory: 2,
      gpuModel: "",
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

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
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
              <Button
                variant="outline"
                type="button"
                className="relative h-8 cursor-pointer"
              >
                <Input
                  onChange={(e) => {
                    importFromJson<FormSchema>(e.target.files?.[0])
                      .then((data) => {
                        form.reset(data);
                        if (data.shareDirs.length > 0) {
                          setDataMountOpen(DataMount);
                        }
                      })
                      .catch(() => {
                        toast.error(`解析错误，导入配置失败`);
                      });
                  }}
                  type="file"
                  className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
                />
                <CircleArrowDown className="-ml-0.5 mr-1.5 h-4 w-4" />
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
                      isValid &&
                        exportToJson(
                          currentValues,
                          currentValues.taskname + ".json",
                        );
                    })
                    .catch((error) => {
                      toast.error((error as Error).message);
                    });
                }}
              >
                <CircleArrowUp className="-ml-0.5 mr-1.5 h-4 w-4" />
                导出配置
              </Button>
              <Button type="submit" className="h-8">
                <CirclePlus className="-ml-0.5 mr-1.5 h-4 w-4" />
                提交作业
              </Button>
            </div>
          </div>
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>基本设置</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-5">
              <div className="grid grid-cols-3 gap-3">
                <div className="col-span-3">
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
              </div>
              <FormField
                control={form.control}
                name="gpuModel"
                render={({ field }) => (
                  <FormItem
                  // hidden={currentValues.gpu == 0}
                  >
                    <FormLabel>
                      节点类型
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Combobox
                        items={labelsInfo.data?.list ?? []}
                        current={field.value}
                        handleSelect={(value) => field.onChange(value)}
                        formTitle="节点类型"
                      />
                    </FormControl>
                    <FormDescription>
                      如果不需要 GPU，选择默认即可
                    </FormDescription>
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
            <AccordionCard cardTitle="环境变量">TODO</AccordionCard>
            <AccordionCard
              cardTitle={DataMount}
              value={dataMountOpen}
              setValue={setDataMountOpen}
            >
              <div className="mt-3 space-y-5">
                {shareDirsFields.map((field, index) => (
                  <div key={field.id}>
                    <Separator
                      className={cn("mb-5", index === 0 && "hidden")}
                    />
                    <div className="space-y-5">
                      <FormField
                        control={form.control}
                        name={`shareDirs.${index}.subPath`}
                        render={({ field }) => (
                          <FormItem className="relative">
                            <FormLabel>
                              挂载源 {index + 1}
                              <FormLabelMust />
                            </FormLabel>
                            <button
                              onClick={() => shareDirsRemove(index)}
                              className="absolute -top-1.5 right-0 rounded-sm opacity-70 transition-opacity hover:opacity-100 focus:outline-none disabled:pointer-events-none data-[state=open]:bg-accent data-[state=open]:text-muted-foreground"
                            >
                              <Cross2Icon className="h-4 w-4" />
                              <span className="sr-only">Close</span>
                            </button>
                            <FormControl>
                              <Tabs defaultValue="file" className="w-full">
                                <TabsList className="grid w-full grid-cols-2">
                                  <TabsTrigger value="file">
                                    文件系统
                                  </TabsTrigger>
                                  <TabsTrigger value="dataset">
                                    数据集
                                  </TabsTrigger>
                                </TabsList>
                                <TabsContent value="file">
                                  <FileSelectDialog
                                    value={field.value.split("/").pop()}
                                    handleSubmit={(item) => {
                                      field.onChange(item.id);
                                      form.setValue(
                                        `shareDirs.${index}.mountPath`,
                                        `/mnt/${item.name}`,
                                      );
                                    }}
                                  />
                                </TabsContent>
                                <TabsContent value="dataset">
                                  <Input disabled />
                                </TabsContent>
                              </Tabs>
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`shareDirs.${index}.mountPath`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              挂载点 {index + 1}
                              <FormLabelMust />
                            </FormLabel>
                            <FormControl>
                              <Input {...field} />
                            </FormControl>
                            <FormDescription>
                              挂载到容器中的路径
                            </FormDescription>
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
                    shareDirsAppend({
                      subPath: "",
                      mountPath: "",
                    })
                  }
                >
                  <CirclePlus className="-ml-0.5 mr-1.5 h-4 w-4" />
                  添加挂载数据
                </Button>
              </div>
            </AccordionCard>
          </div>
        </form>
      </Form>
    </>
  );
};

export default JupyterNew;
