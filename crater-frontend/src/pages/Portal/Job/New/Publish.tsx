import { useState } from "react";
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
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import SandwichSheet from "@/components/sheet/SandwichSheet";
import FormLabelMust from "@/components/form/FormLabelMust";
import TooltipButton from "@/components/custom/TooltipButton";
import { createJobTemplate } from "@/services/api/jobtemplate";
import { toast } from "sonner";

// Define the form schema with Zod
const formSchema = z.object({
  name: z.string().min(1, { message: "Name is required" }),
  description: z.string().optional(),
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

  const [isOpen, setIsOpen] = useState(false);
  // 解析为 JSON 对象
  const objconfig = typeof config === "string" ? JSON.parse(config) : config;
  // 合并到新对象
  const objcombinedConfig = {
    ...objconfig,
    data, // 将 B 作为 data 属性
  };

  // 生成字符串 C（带格式化缩进）
  const formattedConfig = JSON.stringify(objcombinedConfig, null, 2);
  const { mutate: createTemplate } = useMutation({
    mutationFn: (values: FormValues) =>
      createJobTemplate({
        name: values.name,
        describe: values.description || "",
        template: formattedConfig,
        document: "",
      }),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ["list", "jobtemplate"],
      });
      toast.success(`Job template ${name} created successfully`);
      setIsOpen(false);
    },
  });
  // Initialize react-hook-form
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      description: "",
    },
  });

  const handleSubmit = form.handleSubmit((data) => {
    createTemplate(data);
  });

  return (
    <SandwichSheet
      isOpen={isOpen}
      onOpenChange={setIsOpen}
      title="分享配置 (开发中，暂不可用)"
      description="将此配置文件分享给其他用户"
      trigger={
        <TooltipButton
          variant="outline"
          tooltipContent="与平台用户共享，便于快速复现实验"
        >
          <Share2 />
          分享配置
        </TooltipButton>
      }
      footer={<Button onClick={() => handleSubmit()}>公开此配置文件</Button>}
      className="sm:max-w-2xl"
    >
      <Form {...form}>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4 px-6">
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
                <FormLabel>
                  描述
                  <FormLabelMust />
                </FormLabel>
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

          <div className="space-y-2">
            <Label htmlFor="config-preview">配置预览</Label>
            <div className="relative">
              <Textarea
                id="config-preview"
                value={formattedConfig}
                readOnly
                className="bg-muted font-mono text-sm"
              />
            </div>
            <Label className="text-muted-foreground">
              请确保您不会在此配置中包含任何敏感信息，如密码、密钥等
            </Label>
          </div>
        </form>
      </Form>
    </SandwichSheet>
  );
}
