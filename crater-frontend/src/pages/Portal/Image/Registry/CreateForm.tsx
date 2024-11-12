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
import { Button } from "@/components/ui/button";
import { apiUserCreateKaniko } from "@/services/api/imagepack";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import FormLabelMust from "@/components/custom/FormLabelMust";
import { Textarea } from "@/components/ui/textarea";
import { PackagePlusIcon } from "lucide-react";

const formSchema = z.object({
  gitRepository: z.string().optional(),
  accessToken: z.string().optional(),
  dockerfile: z.string().optional(),
  dockerfileSource: z.enum(["repo", "file"]),
  imageName: z.string().min(1, { message: "镜像名不能为空" }),
  imageTag: z.string().min(1, { message: "标签不能为空" }),
  description: z.string().min(1, { message: "镜像描述不能为空" }),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function ImageCreateForm({ closeSheet }: TaskFormProps) {
  const queryClient = useQueryClient();

  const { mutate: createImagePack } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiUserCreateKaniko({
        gitRepository: values.gitRepository ? values.gitRepository : "",
        accessToken: values.accessToken ? values.accessToken : "",
        dockerfile: values.dockerfile ? values.dockerfile : "",
        registryServer: "",
        registryUser: "",
        registryPass: "",
        registryProject: "",
        imageName: values.imageName,
        imageTag: values.imageTag,
        description: values.description,
      }),
    onSuccess: async (_, { imageName, imageTag }) => {
      await queryClient.invalidateQueries({
        queryKey: ["imagepack", "list"],
      });
      toast.success(`镜像 ${imageName}:${imageTag} 创建成功`);
      closeSheet();
    },
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      dockerfileSource: "file",
      gitRepository: "",
      accessToken: "",
      dockerfile: "",
      imageName: "",
      imageTag: "",
      description: "",
    },
  });

  const currentValues = form.watch();

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    createImagePack(values);
  };

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="mt-6 flex flex-col space-y-4"
      >
        <div className="grid grid-cols-2 gap-3">
          <FormField
            control={form.control}
            name="imageName"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  镜像名称
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
            name="imageTag"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  镜像标签
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
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
        <FormField
          control={form.control}
          name="dockerfileSource"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                镜像构建方式
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Select
                  onValueChange={field.onChange}
                  defaultValue={field.value}
                >
                  <SelectTrigger className="">
                    <SelectValue placeholder="请选择镜像构建方式" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="file">Dockerfile</SelectItem>
                    <SelectItem value="repo">代码仓库</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="gitRepository"
          render={({ field }) => (
            <FormItem hidden={currentValues.dockerfileSource == "file"}>
              <FormLabel>
                Git 仓库地址
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="accessToken"
          render={({ field }) => (
            <FormItem hidden={currentValues.dockerfileSource == "file"}>
              <FormLabel>
                Access Token
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="">
          <FormField
            control={form.control}
            name="dockerfile"
            render={({ field }) => (
              <FormItem hidden={currentValues.dockerfileSource == "repo"}>
                <FormLabel>
                  Dockerfile
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Textarea {...field} className="h-24 font-mono" />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
        <div className="flex flex-row justify-end">
          <Button type="submit">
            <PackagePlusIcon />
            提交镜像
          </Button>
        </div>
      </form>
    </Form>
  );
}
