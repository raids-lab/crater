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
import { Textarea } from "@/components/ui/textarea";
import { z } from "zod";
import { toast } from "sonner";
import SandwichSheet, {
  SandwichSheetProps,
} from "@/components/sheet/SandwichSheet";
import LoadableButton from "@/components/custom/LoadableButton";
import { PackagePlusIcon } from "lucide-react";
import FormImportButton from "@/components/form/FormImportButton";
import FormExportButton from "@/components/form/FormExportButton";
import { MetadataFormPipApt } from "@/components/form/types";
import FormLabelMust from "@/components/form/FormLabelMust";
import { JobType } from "@/services/api/vcjob";
import Combobox from "@/components/form/Combobox";
import ImageItem from "@/components/form/ImageItem";
import useImageQuery from "@/hooks/query/useImageQuery";
import { Input } from "@/components/ui/input";
import {
  apiUserCreateKaniko,
  imageNameRegex,
  imageTagRegex,
} from "@/services/api/imagepack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";

export const pipAptFormSchema = z.object({
  baseImage: z.string().min(1, "基础镜像是必填项"),
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
  requirements: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          try {
            v.split("\n").forEach((line) => {
              if (line.trim().startsWith("#")) {
                return;
              }
              if (!line.trim()) {
                return;
              }
              // relation:
              // ==：等于
              // >：大于版本
              // >=：大于等于
              // <：小于版本
              // <=：小于等于版本
              // ~=：兼容版本

              // 基于 relation 将每一行的内容进行分割，分割后的内容为：name, relation, version
              // 可以只有 name
              const regex = /([a-zA-Z0-9_]+)([<>=!~]+)?([a-zA-Z0-9_.]+)?/;
              const match = line.match(regex);
              if (!match) {
                throw new Error("Invalid requirement format");
              }
              if (match.length < 2) {
                throw new Error("Invalid requirement format");
              }
            });
          } catch {
            return false;
          }
        }
        return true;
      },
      {
        message: "requirements.txt 文件格式无效",
      },
    ),
  aptPackages: z.string().optional(),
  description: z.string().min(1, "请为镜像添加描述"),
});

export type PipAptFormValues = z.infer<typeof pipAptFormSchema>;

interface PipAptSheetContentProps {
  form: UseFormReturn<PipAptFormValues>;
  onSubmit: (values: PipAptFormValues) => void;
}

function PipAptSheetContent({ form, onSubmit }: PipAptSheetContentProps) {
  const { data: images } = useImageQuery(JobType.Jupyter);

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 px-6">
      <FormField
        control={form.control}
        name="baseImage"
        render={({ field }) => (
          <FormItem>
            <FormLabel>
              基础镜像
              <FormLabelMust />
            </FormLabel>
            <FormControl autoFocus={true}>
              <Combobox
                items={images ?? []}
                current={field.value}
                handleSelect={(value) => field.onChange(value)}
                renderLabel={(item) => <ImageItem item={item} />}
                formTitle="镜像"
              />
            </FormControl>
            <FormDescription>
              选择一个带有所需 CUDA 和 Python 版本的基础镜像。
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
              <Input
                placeholder="关于此镜像的简短描述，如包含的软件版本、用途等，将作为镜像标识显示。"
                {...field}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="aptPackages"
        render={({ field }) => (
          <FormItem>
            <FormLabel>APT Packages</FormLabel>
            <FormControl>
              <Textarea
                placeholder="输入要安装的 APT 包，例如 git、curl 等。使用空格分隔多个包。"
                className="h-24 font-mono"
                {...field}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="requirements"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Python 依赖</FormLabel>
            <FormControl>
              <Textarea
                placeholder="请粘贴 requirements.txt 文件的内容，以便安装所需的 Python 包。"
                className="h-24 font-mono"
                {...field}
              />
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

export function PipAptSheet({ closeSheet, ...props }: DockerfileSheetProps) {
  const queryClient = useQueryClient();

  const form = useForm<PipAptFormValues>({
    resolver: zodResolver(pipAptFormSchema),
    defaultValues: {
      baseImage: "",
      imageName: "",
      imageTag: "",
      requirements: "",
      aptPackages: "",
      description: "",
    },
  });

  const { mutate: submitDockerfileSheet, isPending } = useMutation({
    mutationFn: (values: PipAptFormValues) =>
      apiUserCreateKaniko({
        description: values.description,
        image: values.baseImage,
        requirements: values.requirements ?? "",
        packages: values.aptPackages ?? "",
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

  const onSubmit = (values: PipAptFormValues) => {
    submitDockerfileSheet(values);
  };

  return (
    <Form {...form}>
      <SandwichSheet
        {...props}
        footer={
          <>
            <FormImportButton metadata={MetadataFormPipApt} form={form} />
            <FormExportButton metadata={MetadataFormPipApt} form={form} />
            <LoadableButton
              isLoading={isPending}
              isLoadingText="正在提交"
              type="submit"
              onClick={async () => {
                // Trigger validations before submitting
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
        <PipAptSheetContent form={form} onSubmit={onSubmit} />
      </SandwichSheet>
    </Form>
  );
}
