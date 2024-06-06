import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage,
  Form,
} from "@/components/ui/form";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

const formSchema = z.object({
  scheduler: z.string().min(1, { message: "请选择调度算法" }),
});

type FormSchema = z.infer<typeof formSchema>;

export const Component = () => {
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      scheduler: "volcano",
    },
  });

  const handleSubmit = () => {
    toast.success("更新成功");
  };

  return (
    <Card className="lg:col-span-2">
      <CardHeader>
        <CardTitle>基本设置</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-5">
        <Form {...form}>
          <form
            // eslint-disable-next-line @typescript-eslint/no-misused-promises
            onSubmit={form.handleSubmit(handleSubmit)}
            className="flex flex-col space-y-4"
          >
            <FormField
              control={form.control}
              name="scheduler"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    调度算法<span className="ml-1 text-red-500">*</span>
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
                        <SelectItem value="original">
                          Kubernetes 原生调度算法
                        </SelectItem>
                        <SelectItem value="volcano">
                          Volcano 调度算法
                        </SelectItem>
                        <SelectItem value="colocate">
                          GPUSched -
                          多租户弹性资源管理和基于性能干扰感知预测模型的作业混合放置算法
                        </SelectItem>
                        <SelectItem value="sprase">
                          RecSched - 面向深度推荐系统训练的资源调度算法
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex flex-row-reverse">
              <Button type="submit">更新设置</Button>
            </div>
          </form>
        </Form>
      </CardContent>
    </Card>
  );
};
