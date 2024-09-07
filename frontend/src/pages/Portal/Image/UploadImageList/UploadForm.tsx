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
import { apiUserImagepackUpload } from "@/services/api/imagepack";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import FormLabelMust from "@/components/custom/FormLabelMust";

const formSchema = z.object({
  imageLink: z.string().min(1, { message: "镜像链接不能为空" }),
  imageName: z.string().min(1, { message: "镜像名不能为空" }),
  imageTag: z.string().min(1, { message: "标签不能为空" }),
  alias: z.string().min(1, { message: "镜像别名不能为空" }),
  description: z.string().min(1, { message: "镜像描述不能为空" }),
  taskType: z.enum(["1", "2"]),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function ImageUploadForm({ closeSheet }: TaskFormProps) {
  const queryClient = useQueryClient();

  const { mutate: uploadImagePack } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiUserImagepackUpload({
        imageLink: values.imageLink,
        imageName: values.imageName,
        imageTag: values.imageTag,
        alias: values.alias,
        description: values.description,
        taskType: Number(values.taskType),
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
      imageLink: "",
      imageName: "",
      imageTag: "",
      alias: "",
      description: "",
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
      <form
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        onSubmit={form.handleSubmit(onSubmit)}
        className="mt-6 flex flex-col space-y-4"
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
        <div>
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
            <FormField
              control={form.control}
              name="alias"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像别名
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
          </div>
        </div>
        <div>
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
                      <SelectItem value="1">离线任务</SelectItem>
                      <SelectItem value="2">Jupyter交互式任务</SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
        <div className="grid">
          <Button type="submit">提交镜像</Button>
        </div>
      </form>
    </Form>
  );
}
