import { Button } from "@/components/ui/button";
import {
  CalendarIcon,
  CircleArrowDown,
  CircleArrowUp,
  CirclePlusIcon,
  XIcon,
} from "lucide-react";
import { useEffect, useState } from "react";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { useFieldArray, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { logger } from "@/utils/loglevel";
import FormLabelMust from "@/components/custom/FormLabelMust";
import LoadableButton from "@/components/custom/LoadableButton";
import { z } from "zod";
import { toast } from "sonner";
import { ScrollArea } from "@/components/ui/scroll-area";
import useResizeObserver from "use-resize-observer";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { exportToJson, importFromJson } from "@/utils/form";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { Calendar } from "@/components/ui/calendar";
import { format } from "date-fns";

const VERSION = "20240623";
const JOB_TYPE = "queue";

const resourceSchema = z.array(
  z.object({
    name: z.string().min(1, {
      message: "资源名称不能为空",
    }),
    guaranteed: z
      .union([
        z.string().min(0, {
          message: "资源不能为空",
        }),
        z.number().int().min(0, {
          message: "资源至少为0",
        }),
      ])
      .default("null"),
    deserved: z
      .union([
        z.string().min(0, {
          message: "资源不能为空",
        }),
        z.number().int().min(0, {
          message: "资源至少为0",
        }),
      ])
      .default("null"),
    capacity: z
      .union([
        z.string().min(0, {
          message: "资源不能为空",
        }),
        z.number().int().min(0, {
          message: "资源至少为0",
        }),
      ])
      .default("null"),
  }),
);

const formSchema = z.object({
  name: z
    .string()
    .min(1, {
      message: "账户名称不能为空",
    })
    .max(16, {
      message: "账户名称最多16个字符",
    }),
  resources: resourceSchema,
  expiredDay: z.date().optional(),
  admins: z.array(z.string()),
});

type FormSchema = z.infer<typeof formSchema>;

interface Resource {
  name: string;
  guaranteed: number | string;
  deserved: number | string;
  capacity: number | string;
}

type ProjectSheetProps = React.HTMLAttributes<HTMLDivElement> & {
  formData: FormSchema;
  isPending: boolean;
  resourcesData?: Resource[];
  apiProjcet: (form: FormSchema) => void;
};

export const ProjectSheet = ({
  formData,
  isPending,
  resourcesData,
  apiProjcet,
  children,
}: ProjectSheetProps) => {
  const { ref: refRoot, width, height } = useResizeObserver();

  const [open, setOpenchange] = useState(false);
  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: formData,
  });

  const currentValues = form.watch();

  const {
    fields: resourcesFields,
    append: resourcesAppend,
    remove: resourcesRemove,
  } = useFieldArray<FormSchema>({
    name: "resources",
    control: form.control,
  });

  useEffect(() => {
    if (!open) return;
    if (!resourcesData) return;
    form.reset();
    const resourcesList = resourcesData;
    resourcesAppend(resourcesList);
  }, [form, open, resourcesAppend, resourcesData]);

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    !isPending && apiProjcet(values);
    setOpenchange(false);
  };

  return (
    <Sheet open={open} onOpenChange={setOpenchange}>
      <SheetTrigger asChild>{children}</SheetTrigger>
      {/* scroll in sheet: https://github.com/shadcn-ui/ui/issues/16 */}
      <SheetContent className="flex flex-col gap-6 px-0 sm:max-w-4xl">
        <SheetHeader>
          <SheetTitle className="pl-6">新建账户</SheetTitle>
        </SheetHeader>

        <div ref={refRoot} className="h-full w-full">
          <ScrollArea style={{ width, height }}>
            <Form {...form}>
              <form
                // eslint-disable-next-line @typescript-eslint/no-misused-promises
                onSubmit={form.handleSubmit(onSubmit)}
                className="grid gap-4 px-6"
              >
                <div className="flex flex-row items-end justify-between gap-4">
                  <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                      <FormItem className="col-span-1 flex-grow">
                        <FormLabel>
                          账户名称
                          <FormLabelMust />
                        </FormLabel>
                        <FormControl>
                          <Input
                            autoComplete="off"
                            {...field}
                            className="w-full"
                          />
                        </FormControl>
                        <FormDescription>
                          账户名称最多16个字符，可以包含汉字，但必须是唯一的
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="expiredDay"
                    render={({ field }) => (
                      <FormItem className="col-span-1 flex flex-col space-y-2">
                        <FormLabel className="pb-1">过期时间</FormLabel>
                        <Popover>
                          <PopoverTrigger asChild>
                            <FormControl>
                              <Button
                                variant={"outline"}
                                className={cn(
                                  "w-[240px] pl-3 text-left font-normal",
                                  !field.value && "text-muted-foreground",
                                )}
                              >
                                {field.value ? (
                                  format(field.value, "PPP")
                                ) : (
                                  <span>请选择日期</span>
                                )}
                                <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                              </Button>
                            </FormControl>
                          </PopoverTrigger>
                          <PopoverContent className="w-auto p-0" align="start">
                            <Calendar
                              mode="single"
                              selected={field.value}
                              onSelect={field.onChange}
                              disabled={(date) => date < new Date()}
                              initialFocus
                            />
                          </PopoverContent>
                        </Popover>
                        <FormDescription>
                          若不填写，账户将永不过期
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
                <div className="space-y-2">
                  {resourcesFields.length > 0 && (
                    <FormLabel>资源配额</FormLabel>
                  )}
                  {resourcesFields.map(({ id }, index) => (
                    <div key={id} className="flex flex-row gap-2">
                      <FormField
                        control={form.control}
                        name={`resources.${index}.name`}
                        render={({ field }) => (
                          <FormItem className="w-fit">
                            <FormControl>
                              <Input
                                {...field}
                                placeholder="资源"
                                className="font-mono"
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`resources.${index}.guaranteed`}
                        render={() => (
                          <FormItem>
                            <FormControl>
                              <Input
                                type="string"
                                placeholder="保证"
                                className="font-mono"
                                {...form.register(
                                  `resources.${index}.guaranteed`,
                                )}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`resources.${index}.deserved`}
                        render={() => (
                          <FormItem>
                            <FormControl>
                              <Input
                                type="string"
                                placeholder="应得"
                                className="font-mono"
                                {...form.register(
                                  `resources.${index}.deserved`,
                                )}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`resources.${index}.capacity`}
                        render={() => (
                          <FormItem>
                            <FormControl>
                              <Input
                                type="string"
                                placeholder="上限"
                                className="font-mono"
                                {...form.register(
                                  `resources.${index}.capacity`,
                                )}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <div>
                        <Button
                          size="icon"
                          type="button"
                          variant="outline"
                          onClick={() => resourcesRemove(index)}
                        >
                          <XIcon className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() =>
                      resourcesAppend({
                        name: "",
                        guaranteed: "",
                        deserved: "",
                        capacity: "",
                      })
                    }
                  >
                    <CirclePlusIcon className="mr-2 h-4 w-4" />
                    添加配额维度
                  </Button>
                </div>
              </form>
            </Form>
          </ScrollArea>
        </div>
        <SheetFooter className="px-6">
          <Button
            variant="outline"
            type="button"
            className="relative cursor-pointer"
          >
            <Input
              onChange={(e) => {
                importFromJson<FormSchema>(
                  VERSION,
                  JOB_TYPE,
                  e.target.files?.[0],
                )
                  .then((data) => {
                    // preserve the name
                    const name = currentValues.name;
                    form.reset(data);
                    form.setValue("name", name);
                    toast.success(`导入配置成功`);
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
            onClick={() => {
              form
                .trigger()
                .then((isValid) => {
                  isValid &&
                    exportToJson(
                      {
                        version: VERSION,
                        type: JOB_TYPE,
                        data: currentValues,
                      },
                      currentValues.name + ".json",
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
          <LoadableButton
            isLoading={isPending}
            type="submit"
            onClick={() => {
              form
                .trigger()
                .then(() => {
                  if (form.formState.isValid) {
                    onSubmit(form.getValues());
                  }
                })
                .catch((e) => logger.debug(e));
            }}
          >
            <CirclePlusIcon className="-ml-0.5 mr-1.5 h-4 w-4" />
            提交申请
          </LoadableButton>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
};
