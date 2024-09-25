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
import {
  apiUserImagepackUpload,
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
import FormLabelMust from "@/components/custom/FormLabelMust";

const formSchema = z.object({
  imageLink: z
    .string()
    .min(1, { message: "镜像链接不能为空" })
    .refine((value) => imageLinkRegex.test(value), {
      message: "镜像链接格式不正确",
    }),
  alias: z.string().min(1, { message: "镜像别名不能为空" }),
  description: z.string().min(1, { message: "镜像描述不能为空" }),
  taskType: z.enum(["1", "2", "3", "4", "5", "6", "7", "8"]),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function ImageUploadForm({ closeSheet }: TaskFormProps) {
  const queryClient = useQueryClient();

  const { mutate: uploadImagePack } = useMutation({
    mutationFn: (values: FormSchema) => {
      const { imageName, imageTag } = parseImageLink(values.imageLink);
      return apiUserImagepackUpload({
        imageLink: values.imageLink,
        imageName: imageName,
        imageTag: imageTag,
        alias: values.alias,
        description: values.description,
        taskType: Number(values.taskType),
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
                      <SelectItem value="1">Jupyter交互式任务</SelectItem>
                      <SelectItem value="2" disabled>
                        Web IDE任务
                      </SelectItem>
                      <SelectItem value="3">Tensorflow任务</SelectItem>
                      <SelectItem value="4">Pytorch任务</SelectItem>
                      <SelectItem value="5" disabled>
                        Ray任务
                      </SelectItem>
                      <SelectItem value="6" disabled>
                        DeepSpeed任务
                      </SelectItem>
                      <SelectItem value="7" disabled>
                        OpenMPI任务
                      </SelectItem>
                      <SelectItem value="8">用户自定义任务</SelectItem>
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
