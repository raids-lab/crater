import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  apiUserUploadImage,
  ImageDefaultTags,
  imageLinkRegex,
  parseImageLink,
} from "@/services/api/imagepack";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import FormLabelMust from "@/components/form/FormLabelMust";
import { JobType } from "@/services/api/vcjob";
import SandwichSheet, {
  SandwichLayout,
  SandwichSheetProps,
} from "@/components/sheet/SandwichSheet";
import LoadableButton from "@/components/custom/LoadableButton";
import { PackagePlusIcon } from "lucide-react";
import { TagsInput } from "@/components/form/TagsInput";

const formSchema = z.object({
  imageLink: z
    .string()
    .min(1, { message: "镜像链接不能为空" })
    .refine((value) => imageLinkRegex.test(value), {
      message: "镜像链接格式不正确",
    }),
  description: z.string().min(1, { message: "镜像描述不能为空" }),
  taskType: z
    .enum([
      JobType.Custom,
      JobType.DeepSpeed,
      JobType.Jupyter,
      JobType.KubeRay,
      JobType.OpenMPI,
      JobType.Pytorch,
      JobType.Tensorflow,
    ])
    .refine((value) => Object.values(JobType).includes(value), {
      message: "请选择任务类型",
    }),
  tags: z
    .array(
      z.object({
        value: z.string(),
      }),
    )
    .optional(),
});

type FormSchema = z.infer<typeof formSchema>;

interface ImageUploadSheetContentProps {
  closeSheet: () => void;
}

function ImageUploadSheetContent({ closeSheet }: ImageUploadSheetContentProps) {
  const queryClient = useQueryClient();

  const { mutate: uploadImagePack, isPending } = useMutation({
    mutationFn: (values: FormSchema) => {
      const { imageName, imageTag } = parseImageLink(values.imageLink);
      return apiUserUploadImage({
        imageLink: values.imageLink,
        imageName: imageName,
        imageTag: imageTag,
        description: values.description,
        taskType: values.taskType,
        tags: values.tags?.map((item) => item.value) ?? [],
      });
    },
    onSuccess: async (_, { imageLink }) => {
      await queryClient.invalidateQueries({
        queryKey: ["imagelink", "list"],
      });
      toast.success(`镜像 ${imageLink} 上传成功`);
      closeSheet();
    },
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      imageLink: "",
      description: "",
      tags: [],
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    uploadImagePack(values);
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 px-6">
        <SandwichLayout
          footer={
            <LoadableButton
              isLoading={isPending}
              isLoadingText="正在提交"
              type="submit"
            >
              <PackagePlusIcon />
              确认提交
            </LoadableButton>
          }
        >
          <FormField
            control={form.control}
            name="imageLink"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  镜像链接
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" />
                </FormControl>
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
                  镜像描述
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" />
                </FormControl>
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
            name="taskType"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  任务类型
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Select
                    onValueChange={field.onChange}
                    defaultValue={field.value}
                  >
                    <SelectTrigger className="">
                      <SelectValue placeholder="请选择" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={JobType.Jupyter}>
                        Jupyter交互式任务
                      </SelectItem>

                      <SelectItem value={JobType.Tensorflow}>
                        Tensorflow任务
                      </SelectItem>
                      <SelectItem value={JobType.Pytorch}>
                        Pytorch任务
                      </SelectItem>
                      <SelectItem value={JobType.Custom}>
                        用户自定义任务
                      </SelectItem>
                      <SelectItem value={JobType.KubeRay} disabled>
                        Ray任务
                      </SelectItem>
                      <SelectItem value={JobType.DeepSpeed} disabled>
                        DeepSpeed任务
                      </SelectItem>
                      <SelectItem value={JobType.OpenMPI} disabled>
                        OpenMPI任务
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </SandwichLayout>
      </form>
    </Form>
  );
}
interface ImageUploadFormProps extends SandwichSheetProps {
  closeSheet: () => void;
}

export function ImageUploadForm({
  closeSheet,
  ...props
}: ImageUploadFormProps) {
  return (
    <SandwichSheet {...props}>
      <ImageUploadSheetContent closeSheet={closeSheet} />
    </SandwichSheet>
  );
}
