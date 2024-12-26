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
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import FormLabelMust from "@/components/form/FormLabelMust";
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
  const onSubmit = () => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.

    // The corresponding interface has been used to create from dockerfile sheets
    closeSheet();
  };

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="flex flex-col space-y-4 px-6"
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
