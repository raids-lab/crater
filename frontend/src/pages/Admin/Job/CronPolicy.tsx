import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { logger } from "@/utils/loglevel";
import * as z from "zod";
import { AlarmClockIcon, FileCogIcon } from "lucide-react";
import TipBadge from "@/components/badge/TipBadge";
import { isValidCron } from "cron-validator";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

const formSchema = z.object({
  notifyEnabled: z.boolean(),
  notifySchedule: z.string().refine((value) => isValidCron(value), {
    message: "无效的 Cron 表达式，请检查格式和数值范围",
  }),
  notifyTimeRange: z.number().int().positive(),
  notifyUtil: z.number().min(0).max(100),
  cleanEnabled: z.boolean(),
  cleanSchedule: z.string().refine((value) => isValidCron(value), {
    message: "无效的 Cron 表达式，请检查格式和数值范围",
  }),
  cleanTimeRange: z.number().int().positive(),
  cleanUtil: z.number().min(0).max(100),
});

type FormValues = z.infer<typeof formSchema>;

export default function CronPolicy({ className }: { className?: string }) {
  const [dryRunJobs, setDryRunJobs] = useState<string[]>([]);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      notifyEnabled: true,
      notifySchedule: "0 */6 * * *",
      notifyTimeRange: 6,
      notifyUtil: 0,
      cleanEnabled: true,
      cleanSchedule: "0 */8 * * *",
      cleanTimeRange: 6,
      cleanUtil: 0,
    },
  });

  const onSubmit = (data: FormValues) => {
    logger.info("Form data", data);
    toast.warning("TODO(zhangry): Implement form submission");
    // Here you would typically send the data to your backend
  };

  const runJob = async () => {
    // This is a mock function. In a real scenario, you'd call your backend API
    const mockDryRunJobs = ["Job1", "Job2", "Job3"];
    setDryRunJobs(mockDryRunJobs);
  };

  const confirmJobRun = async () => {
    // This is where you'd actually run the job
    // Reset dry run jobs
    setDryRunJobs([]);
  };

  return (
    <Card className={cn("flex flex-col justify-between", className)}>
      <CardHeader>
        <CardTitle className="flex flex-row items-center gap-1.5">
          <AlarmClockIcon className="text-primary" />
          定时策略
          <TipBadge />
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
            <div className="space-y-4">
              <FormField
                control={form.control}
                name="notifyEnabled"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                    <FormControl>
                      <Checkbox
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <div className="text-sm">
                      每
                      <FormField
                        control={form.control}
                        name="notifySchedule"
                        render={({ field }) => (
                          <Input
                            className="mx-1 inline-block h-6 w-32 font-mono"
                            {...field}
                          />
                        )}
                      />
                      提醒
                      <FormField
                        control={form.control}
                        name="notifyTimeRange"
                        render={({ field }) => (
                          <Input
                            type="number"
                            className="mx-1 inline-block h-6 w-16 font-mono"
                            {...field}
                            onChange={(e) =>
                              field.onChange(parseInt(e.target.value))
                            }
                          />
                        )}
                      />
                      小时利用率不超过
                      <FormField
                        control={form.control}
                        name="notifyUtil"
                        render={({ field }) => (
                          <Input
                            type="number"
                            className="mx-1 inline-block h-6 w-16 font-mono"
                            {...field}
                            onChange={(e) =>
                              field.onChange(parseFloat(e.target.value))
                            }
                          />
                        )}
                      />
                      的作业
                    </div>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="cleanEnabled"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                    <FormControl>
                      <Checkbox
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <div className="text-sm">
                      每
                      <FormField
                        control={form.control}
                        name="cleanSchedule"
                        render={({ field }) => (
                          <Input
                            className="mx-1 inline-block h-6 w-32 font-mono"
                            {...field}
                          />
                        )}
                      />
                      清理
                      <FormField
                        control={form.control}
                        name="cleanTimeRange"
                        render={({ field }) => (
                          <Input
                            type="number"
                            className="mx-1 inline-block h-6 w-16 font-mono"
                            {...field}
                            onChange={(e) =>
                              field.onChange(parseInt(e.target.value))
                            }
                          />
                        )}
                      />
                      小时利用率不超过
                      <FormField
                        control={form.control}
                        name="cleanUtil"
                        render={({ field }) => (
                          <Input
                            type="number"
                            className="mx-1 inline-block h-6 w-16 font-mono"
                            {...field}
                            onChange={(e) =>
                              field.onChange(parseFloat(e.target.value))
                            }
                          />
                        )}
                      />
                      的作业
                    </div>
                  </FormItem>
                )}
              />
            </div>
          </form>
        </Form>
      </CardContent>
      <CardFooter className="flex justify-start gap-3">
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button onClick={runJob} variant="destructive">
              {/* https://lucide.dev/icons/lab/broom */}
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                className="lucide lucide-broom"
              >
                <path d="m13 11 9-9" />
                <path d="M14.6 12.6c.8.8.9 2.1.2 3L10 22l-8-8 6.4-4.8c.9-.7 2.2-.6 3 .2Z" />
                <path d="m6.8 10.4 6.8 6.8" />
                <path d="m5 17 1.4-1.4" />
              </svg>
              立即清理
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>作业清理确认</AlertDialogTitle>
              <AlertDialogDescription>
                {dryRunJobs.length > 0 ? (
                  <>
                    以下作业将被删除：
                    {dryRunJobs.map((job, index) => (
                      <div key={index}>{job}</div>
                    ))}
                  </>
                ) : (
                  "没有符合条件的作业需要删除。"
                )}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction onClick={confirmJobRun}>
                确认
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
        <Button
          type="submit"
          variant="secondary"
          onClick={form.handleSubmit(onSubmit)}
        >
          <FileCogIcon />
          更新策略
        </Button>
      </CardFooter>
    </Card>
  );
}
