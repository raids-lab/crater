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
import { useAtom } from "jotai";
import { globalSettings } from "@/utils/store";
import FormLabelMust from "@/components/custom/FormLabelMust";

const formSchema = z.object({
  scheduler: z.enum(["volcano", "colocate", "sparse"], {
    invalid_type_error: "请选择调度算法",
    required_error: "请选择调度算法",
  }),
});

type FormSchema = z.infer<typeof formSchema>;

export const Component = () => {
  const [settings, setSettings] = useAtom(globalSettings);

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: settings,
  });

  const handleSubmit = () => {
    toast.success("更新成功");
    setSettings(form.getValues());
    // refresh page
    window.location.reload();
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
                    调度算法
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
                        <SelectItem value="volcano">
                          BASE - 原生调度算法
                        </SelectItem>
                        <SelectItem value="colocate">
                          EMIAS - 面向多租户可抢占异构资源场景的调度算法
                        </SelectItem>
                        <SelectItem value="sparse">
                          SEACS - 面向混合机器学习任务的调度算法
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
