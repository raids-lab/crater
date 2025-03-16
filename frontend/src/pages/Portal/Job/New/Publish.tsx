import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
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

// Define the form schema with Zod
const formSchema = z.object({
  name: z.string().min(1, { message: "Name is required" }),
  description: z.string().optional(),
});

type FormValues = z.infer<typeof formSchema>;

interface PublishConfigFormProps {
  config: object; // The configuration object to be published
  onPublish: (name: string, description: string, config: object) => void;
}

export function PublishConfigForm({
  config,
  onPublish,
}: PublishConfigFormProps) {
  const [isOpen, setIsOpen] = useState(false);

  // Initialize react-hook-form
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      description: "",
    },
  });

  const handleSubmit = form.handleSubmit((data) => {
    onPublish(data.name, data.description || "", config);
    setIsOpen(false);
    form.reset();
  });

  const formattedConfig = JSON.stringify(config, null, 2);

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
