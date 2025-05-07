import { useState, useEffect } from "react"; // 修改这一行添加 useEffect
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { FieldValues, UseFormReturn } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Share2 } from "lucide-react";
import { getJobTemplate } from "@/services/api/jobtemplate";
import { useQuery } from "@tanstack/react-query";

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import SandwichSheet, {
  SandwichLayout,
} from "@/components/sheet/SandwichSheet";
import FormLabelMust from "@/components/form/FormLabelMust";
import TooltipButton from "@/components/custom/TooltipButton";
import {
  createJobTemplate,
  updateJobTemplate,
} from "@/services/api/jobtemplate";
import { toast } from "sonner";
import { useSearchParams } from "react-router-dom";
import { globalUserInfo } from "@/utils/store";
import { useAtomValue } from "jotai";
import { CodeContent } from "@/components/codeblock/ConfigDialog";

// Define the form schema with Zod
const formSchema = z.object({
  name: z.string().min(1, { message: "Name is required" }),
  description: z.string().optional(),
  document: z.string().optional(),
});

type FormValues = z.infer<typeof formSchema>;

interface PublishConfigFormProps<T extends FieldValues> {
  config: object; // The configuration object to be published
  configform: UseFormReturn<T>;
}

export function PublishConfigForm<T extends FieldValues>({
  config,
  configform,
}: PublishConfigFormProps<T>) {
  const data = configform?.getValues();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const user = useAtomValue(globalUserInfo);
  const fromTemplate = searchParams.get("fromTemplate");

  const [isOpen, setIsOpen] = useState(false);
  const { data: templateData } = useQuery({
    queryKey: ["jobTemplate", fromTemplate],
    queryFn: () => getJobTemplate(Number(fromTemplate)),
    enabled: !!fromTemplate,
    select: (data) => data?.data?.data || null,
  });
  // 解析为 JSON 对象
  const objconfig = typeof config === "string" ? JSON.parse(config) : config;
  // 合并到新对象
  const objcombinedConfig = {
    ...objconfig,
    data, // 将 B 作为 data 属性
  };
  const isUpdate =
    fromTemplate && templateData?.userInfo.username === user.name;
  // 生成字符串 C（带格式化缩进）
  const formattedConfig = JSON.stringify(objcombinedConfig, null, 2);
  const { mutate: createTemplate } = useMutation({
    mutationFn: (values: FormValues) =>
      isUpdate
        ? updateJobTemplate({
            id: Number(fromTemplate),
            name: values.name,
            describe: values.description || "",
            template: formattedConfig,
            document: values.document || "",
          })
        : createJobTemplate({
            name: values.name,
            describe: values.description || "",
            template: formattedConfig,
            document: values.document || "",
          }),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ["data", "jobtemplate"],
      });
      if (isUpdate) {
        toast.success(`Job template ${name} updated successfully`);
      } else {
        toast.success(`Job template ${name} created successfully`);
      }
      setIsOpen(false);
    },
  });
  // Initialize react-hook-form
  // 提取默认值逻辑，避免重复的三元表达式
  const defaultValues =
    isUpdate && templateData
      ? {
          name: templateData.name || "",
          description: templateData.describe || "",
          document: templateData.document || "",
        }
      : {
          name: "",
          description: "",
          document: "",
        };

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues,
  });
  useEffect(() => {
    if (templateData) {
      form.reset({
        name: templateData.name || "",
        description: templateData.describe || "",
        document: templateData.document || "",
      });
    }
  }, [templateData, form]);

  const handleSubmit = form.handleSubmit((data) => {
    createTemplate(data);
  });

  return (
    <SandwichSheet
      isOpen={isOpen}
      onOpenChange={setIsOpen}
      title={isUpdate ? "更新配置" : "分享配置"}
      description={isUpdate ? "更新此配置文件" : "将此配置文件分享给其他用户"}
      trigger={
        <TooltipButton
          variant="outline"
          tooltipContent={
            isUpdate ? "更新已分享的配置" : "与平台用户共享，便于快速复现实验"
          }
        >
          <Share2 />
          {isUpdate ? "更新配置" : "分享配置"}
        </TooltipButton>
      }
      className="sm:max-w-2xl"
    >
      <Form {...form}>
        <form onSubmit={handleSubmit}>
          <SandwichLayout
            footer={
              <Button
                onClick={async () => {
                  const isValid = await configform.trigger();
                  if (isValid) {
                    handleSubmit();
                  } else {
                    toast.error("请检查作业配置是否填写正确");
                  }
                }}
              >
                {isUpdate ? "更新此配置文件" : "公开此配置文件"}
              </Button>
            }
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    配置名称
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormDescription>
                    给此配置文件起一个名字，将作为配置文件的标题显示
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>描述</FormLabel>
                  <FormControl>
                    <Textarea {...field} />
                  </FormControl>
                  <FormDescription>
                    关于此配置的简短描述，如包含的软件版本、用途等，将作为配置标识显示
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="document"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>作业模板详细文档</FormLabel>
                  <FormControl>
                    <Textarea {...field} className="min-h-[200px]" />
                  </FormControl>
                  <FormDescription>
                    关于此配置的详细说明，支持 Markdown 格式
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="space-y-2">
              <Label htmlFor="config-preview">配置预览</Label>
              <CodeContent
                data={formattedConfig}
                language="json"
                className="max-w-full"
              />
              <Label className="text-muted-foreground font-normal">
                请确保您不会在此配置中包含任何敏感信息，如密码、密钥等
              </Label>
            </div>
          </SandwichLayout>
        </form>
      </Form>
    </SandwichSheet>
  );
}
