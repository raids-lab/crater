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
import { Cross2Icon, PlusCircledIcon } from "@radix-ui/react-icons";
import { getAiKResource } from "@/utils/resource";
import { toast } from "sonner";
import { ChevronLeftIcon, FileDown, FileUp } from "lucide-react";
import FormLabelMust from "@/components/custom/FormLabelMust";
import Combobox from "@/components/form/Combobox";
import { apiNodeLabelsList } from "@/services/api/nodelabel";

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
  gpuModel: z.string().min(1, {
    message: "请选择节点 GPU 类型",
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
      toast.success(`任务 ${taskname} 创建成功`);
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
              <Button
                variant="outline"
                onClick={loadFromJson}
                type="button"
                className="h-8"
              >
                <FileDown className="-ml-0.5 mr-1.5 h-4 w-4" />
                导入配置
              </Button>
              <Button variant="outline" type="button" className="h-8">
                <FileUp className="-ml-0.5 mr-1.5 h-4 w-4" />
                导出配置
              </Button>
              <Button type="submit" className="h-8">
                <PlusCircledIcon className="-ml-0.5 mr-1.5 h-4 w-4" />
                提交任务
              </Button>
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
                      镜像地址
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
                  <PlusCircledIcon className="-ml-0.5 mr-1.5 h-4 w-4" />
                  添加需要挂载的文件
                </Button>
              </div>
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
