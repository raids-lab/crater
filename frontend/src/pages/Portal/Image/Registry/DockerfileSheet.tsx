import { useForm, UseFormReturn } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { z } from "zod";
import { toast } from "sonner";
import SandwichSheet, {
  SandwichSheetProps,
} from "@/components/sheet/SandwichSheet";
import LoadableButton from "@/components/custom/LoadableButton";
import { PackagePlusIcon } from "lucide-react";
import FormImportButton from "@/components/form/FormImportButton";
import FormExportButton from "@/components/form/FormExportButton";
import { MetadataFormDockerfile } from "@/components/form/types";
import { Input } from "@/components/ui/input";
import {
  apiUserCreateByDockerfile,
  imageNameRegex,
  imageTagRegex,
} from "@/services/api/imagepack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { DockerfileEditor } from "./DockerfileEditor";
import FormLabelMust from "@/components/form/FormLabelMust";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";

export const dockerfileFormSchema = z.object({
  dockerfile: z.string().min(1, "Dockerfile content is required"),
  description: z.string().min(1, "请为镜像添加描述"),
  imageName: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          return imageNameRegex.test(v);
        } else {
          return true;
        }
      },
      {
        message: "仅允许小写字母、数字、. _ -，且不能以分隔符开头/结尾",
      },
    ),
  imageTag: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          return imageTagRegex.test(v);
        } else {
          return true;
        }
      },
      {
        message: "仅允许字母、数字、_ . + -，且不能以 . 或 - 开头/结尾",
      },
    ),
});

export type DockerfileFormValues = z.infer<typeof dockerfileFormSchema>;

interface DockerfileSheetContentProps {
  form: UseFormReturn<DockerfileFormValues>;
  onSubmit: (values: DockerfileFormValues) => void;
}

function DockerfileSheetContent({
  form,
  onSubmit,
}: DockerfileSheetContentProps) {
  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 px-6">
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
              <Input
                // placeholder="关于此镜像的简短描述，如包含的软件版本、用途等，将作为镜像标识显示。"
                {...field}
              />
            </FormControl>
            <FormDescription>
              关于此镜像的简短描述，如包含的软件版本、用途等，将作为镜像标识显示。
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="dockerfile"
        render={({ field }) => (
          <FormItem>
            <FormLabel>
              Dockerfile
              <FormLabelMust />
            </FormLabel>
            <FormControl>
              <DockerfileEditor value={field.value} onChange={field.onChange} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <div className="flex items-start gap-4">
        <Accordion
          type="single"
          collapsible
          className="w-full rounded-lg border"
        >
          <AccordionItem value="image-settings" className="border-none">
            <AccordionTrigger className="px-4 py-3 hover:no-underline">
              高级设置（镜像名和标签）
            </AccordionTrigger>
            <AccordionContent className="px-4 pb-4">
              <div className="flex items-start gap-4">
                <div className="flex-1">
                  <FormField
                    control={form.control}
                    name="imageName"
                    render={({ field }) => (
                      <FormItem className="flex h-full flex-col">
                        <FormLabel>镜像名</FormLabel>
                        <FormControl>
                          <Input {...field} placeholder="默认镜像名" />
                        </FormControl>
                        <FormMessage className="min-h-[20px] leading-none" />
                      </FormItem>
                    )}
                  />
                </div>
                <div className="flex-1">
                  <FormField
                    control={form.control}
                    name="imageTag"
                    render={({ field }) => (
                      <FormItem className="flex h-full flex-col">
                        <FormLabel>镜像标签</FormLabel>
                        <FormControl>
                          <Input {...field} placeholder="默认标签" />
                        </FormControl>
                        <FormMessage className="min-h-[20px] leading-none" />
                      </FormItem>
                    )}
                  />
                </div>
              </div>
              <FormDescription className="mt-2">
                输入用户自定义的镜像名和镜像标签，若为空，则由系统自动生成
              </FormDescription>
            </AccordionContent>
          </AccordionItem>
        </Accordion>
      </div>
    </form>
  );
}

interface DockerfileSheetProps extends SandwichSheetProps {
  closeSheet: () => void;
}

export function DockerfileSheet({
  closeSheet,
  ...props
}: DockerfileSheetProps) {
  const queryClient = useQueryClient();

  const form = useForm<DockerfileFormValues>({
    resolver: zodResolver(dockerfileFormSchema),
    defaultValues: {
      dockerfile:
        'FROM node:14\n\nWORKDIR /app\n\nCOPY package*.json ./\n\nRUN npm install\n\nCOPY . .\n\nEXPOSE 3000\n\nCMD ["npm", "start"]',
      description: "",
      imageName: "",
      imageTag: "",
    },
  });

  const { mutate: submitDockerfileSheet, isPending } = useMutation({
    mutationFn: (values: DockerfileFormValues) =>
      apiUserCreateByDockerfile({
        description: values.description,
        dockerfile: values.dockerfile,
        name: values.imageName ?? "",
        tag: values.imageTag ?? "",
      }),
    onSuccess: async () => {
      await new Promise((resolve) => setTimeout(resolve, 500)).then(() =>
        queryClient.invalidateQueries({ queryKey: ["imagepack", "list"] }),
      );
      closeSheet();
      toast.success(`镜像开始制作，请在下方列表中查看制作状态`);
    },
  });

  const onSubmit = (values: DockerfileFormValues) => {
    submitDockerfileSheet(values);
  };

  return (
    <Form {...form}>
      <SandwichSheet
        {...props}
        footer={
          <>
            <FormImportButton metadata={MetadataFormDockerfile} form={form} />
            <FormExportButton metadata={MetadataFormDockerfile} form={form} />

            <LoadableButton
              isLoading={isPending}
              isLoadingText="正在提交"
              type="submit"
              onClick={async () => {
                const isValid = await form.trigger();
                if (isValid) {
                  form.handleSubmit(onSubmit)();
                }
              }}
            >
              <PackagePlusIcon />
              开始制作
            </LoadableButton>
          </>
        }
      >
        <DockerfileSheetContent form={form} onSubmit={onSubmit} />
      </SandwichSheet>
    </Form>
  );
}
