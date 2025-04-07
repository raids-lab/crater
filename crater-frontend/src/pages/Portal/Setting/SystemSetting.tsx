import {
  Card,
  CardTitle,
  CardContent,
  CardFooter,
  CardHeader,
  CardDescription,
} from "@/components/ui/card";
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
  FormControl,
  FormMessage,
  Form,
} from "@/components/ui/form";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { useAtom } from "jotai";
import { globalSettings } from "@/utils/store";
import { FileCogIcon } from "lucide-react";
import WarningAlert from "@/components/custom/WarningAlert";

const formSchema = z.object({
  scheduler: z.enum(["volcano", "colocate", "sparse"], {
    invalid_type_error: "请选择调度算法",
    required_error: "请选择调度算法",
  }),
});

type FormSchema = z.infer<typeof formSchema>;

const SystemSetting = () => {
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
    <>
      <WarningAlert title={"注意"} description={"项目使用，请谨慎操作"} />
      <Card>
        <CardHeader>
          <CardTitle>调度算法</CardTitle>
          <CardDescription>
            为您的工作负载选取最佳资源分配和调度策略
          </CardDescription>
        </CardHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)}>
            <CardContent>
              <FormField
                control={form.control}
                name="scheduler"
                render={({ field }) => (
                  <FormItem>
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
            </CardContent>
            <CardFooter className="px-6 py-4">
              <Button type="submit">
                <FileCogIcon />
                更新调度算法
              </Button>
            </CardFooter>
          </form>
        </Form>
      </Card>
    </>
  );
};

export default SystemSetting;
