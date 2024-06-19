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
import { apiDatasetCreate } from "@/services/api/dataset";
import { FileSelectDialog } from "@/components/custom/FileSelectDialog";

const formSchema = z.object({
  datasetName: z
    .string()
    .min(1, {
      message: "数据集名称不能为空",
    })
    .max(256, {
      message: "数据集名称最多包含 256 个字符",
    }),
  describe: z.string(),
  url: z.string(),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function DatasetCreateForm({ closeSheet }: TaskFormProps) {
  const queryClient = useQueryClient();

  const { mutate: createImagePack } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiDatasetCreate({
        describe: values.describe,
        name: values.datasetName,
        url: values.url,
      }),
    onSuccess: async (_, { datasetName }) => {
      await queryClient.invalidateQueries({
        queryKey: ["data", "mydataset"],
      });
      toast.success(`数据集 ${datasetName} 创建成功`);
      closeSheet();
    },
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      datasetName: "",
      url: "",
      describe: "",
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    createImagePack(values);
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
          name="datasetName"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                数据集名称<span className="ml-1 text-red-500">*</span>
              </FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              {/* <FormMessage>请输入仓库地址</FormMessage> */}
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="describe"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                数据集描述<span className="ml-1 text-red-500">*</span>
              </FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <div>
          <div className="grid grid-cols-2 gap-3">
            <FormField
              control={form.control}
              name="url"
              render={({ field }) => (
                <FormItem className="col-span-2">
                  <FormLabel>
                    数据集地址<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <FileSelectDialog
                      value={field.value.split("/").pop()}
                      handleSubmit={(item) => {
                        field.onChange(item.id);
                        form.setValue("url", item.id);
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>
        <div className="grid">
          <Button type="submit">提交数据集</Button>
        </div>
      </form>
    </Form>
  );
}
