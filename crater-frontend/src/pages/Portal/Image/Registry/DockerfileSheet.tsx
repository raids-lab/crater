import { useForm } from "react-hook-form";
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
  SandwichLayout,
  SandwichSheetProps,
} from "@/components/sheet/SandwichSheet";
import LoadableButton from "@/components/button/LoadableButton";
import { PackagePlusIcon } from "lucide-react";
import FormImportButton from "@/components/form/FormImportButton";
import FormExportButton from "@/components/form/FormExportButton";
import { MetadataFormDockerfile } from "@/components/form/types";
import { Input } from "@/components/ui/input";
import {
  apiUserCreateByDockerfile,
  dockerfileImageLinkRegex,
  ImageDefaultTags,
  imageNameRegex,
  ImagePackSource,
  imageTagRegex,
} from "@/services/api/imagepack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { DockerfileEditor } from "./DockerfileEditor";
import FormLabelMust from "@/components/form/FormLabelMust";
import { ImageSettingsFormCard } from "@/components/form/ImageSettingsFormCard";
import { TagsInput } from "@/components/form/TagsInput";
import { exportToJsonString } from "@/utils/form";
import { useImageTemplateLoader } from "@/hooks/useTemplateLoader";

const dockerfileFormSchema = z.object({
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
  tags: z
    .array(
      z.object({
        value: z.string(),
      }),
    )
    .optional(),
});

export type DockerfileFormValues = z.infer<typeof dockerfileFormSchema>;

interface DockerfileSheetContentProps {
  closeSheet: () => void;
  imagePackName?: string;
  setImagePackName: (imagePackName: string) => void;
}

function DockerfileSheetContent({
  closeSheet,
  imagePackName = "",
  setImagePackName,
}: DockerfileSheetContentProps) {
  const queryClient = useQueryClient();

  const form = useForm<DockerfileFormValues>({
    resolver: zodResolver(dockerfileFormSchema),
    defaultValues: {
      dockerfile:
        'FROM node:14\n\nWORKDIR /app\n\nCOPY package*.json ./\n\nRUN npm install\n\nCOPY . .\n\nEXPOSE 3000\n\nCMD ["npm", "start"]',
      description: "",
      imageName: "",
      imageTag: "",
      tags: [],
    },
  });

  const { mutate: submitDockerfileSheet, isPending } = useMutation({
    mutationFn: (values: DockerfileFormValues) =>
      apiUserCreateByDockerfile({
        description: values.description,
        dockerfile: values.dockerfile,
        name: values.imageName ?? "",
        tag: values.imageTag ?? "",
        tags: values.tags?.map((item) => item.value) ?? [],
        template: exportToJsonString(MetadataFormDockerfile, values),
        buildSource: ImagePackSource.Dockerfile,
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
    if (isPending) {
      toast.error("正在提交，请稍后");
      return;
    }
    const matches = Array.from(
      values.dockerfile.matchAll(dockerfileImageLinkRegex),
    );
    const baseImages = matches.map((match) => match[1]);
    const baseImage = baseImages[0];
    if (
      baseImage.includes("jupyter") &&
      values.imageName != "" &&
      !values.imageName?.includes("jupyter")
    ) {
      toast.error("基础镜像为 Jupyter 相关镜像时，镜像名称必须包含 jupyter");
      return;
    }
    if (
      baseImage.includes("nvidia") &&
      values.imageName != "" &&
      !values.imageName?.includes("nvidia")
    ) {
      toast.error("基础镜像为 NVIDIA 相关镜像时，镜像名称必须包含 nvidia");
      return;
    }
    submitDockerfileSheet(values);
  };

  useImageTemplateLoader({
    form: form,
    metadata: MetadataFormDockerfile,
    imagePackName: imagePackName,
    setImagePackName: setImagePackName,
  });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SandwichLayout
          footer={
            <>
              <FormImportButton metadata={MetadataFormDockerfile} form={form} />
              <FormExportButton metadata={MetadataFormDockerfile} form={form} />
              <LoadableButton
                isLoading={isPending}
                isLoadingText="正在提交"
                type="submit"
              >
                <PackagePlusIcon />
                开始制作
              </LoadableButton>
            </>
          }
        >
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
                  <Input {...field} />
                </FormControl>
                <FormDescription>
                  关于此镜像的简短描述，如包含的软件版本、用途等，将作为镜像标识显示。
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <TagsInput
            form={form}
            tagsPath="tags"
            label={`镜像标签`}
            description={`为镜像添加标签，以便分类和搜索`}
            customTags={ImageDefaultTags}
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
                  <DockerfileEditor
                    value={field.value}
                    onChange={field.onChange}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <ImageSettingsFormCard
            form={form}
            imageNamePath="imageName"
            imageTagPath="imageTag"
            description="输入用户自定义的镜像名和镜像标签，若为空，则由系统自动生成"
          />
        </SandwichLayout>
      </form>
    </Form>
  );
}

interface DockerfileSheetProps extends SandwichSheetProps {
  closeSheet: () => void;
  imagePackName?: string;
  setImagePackName: (imagePackName: string) => void;
}

export function DockerfileSheet({
  closeSheet,
  imagePackName = "",
  setImagePackName,
  ...props
}: DockerfileSheetProps) {
  return (
    <SandwichSheet {...props}>
      <DockerfileSheetContent
        imagePackName={imagePackName}
        setImagePackName={setImagePackName}
        closeSheet={closeSheet}
      />
    </SandwichSheet>
  );
}
