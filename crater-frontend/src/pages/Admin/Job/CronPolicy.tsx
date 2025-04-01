import { useEffect, useState } from "react";
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
import * as z from "zod";
import { AlarmClockIcon } from "lucide-react";
import TipBadge from "@/components/badge/TipBadge";
import { isValidCron } from "cron-validator";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import {
  apiJobScheduleAdmin,
  apiJobScheduleChangeAdmin,
} from "@/services/api/vcjob";
import { Skeleton } from "@/components/ui/skeleton";

interface Job {
  name: string;
  suspend?: boolean;
  schedule?: string;
  configs?: {
    [key: string]: string;
  };
}

// 将配置项置于 configs 对象下
const cleanLongTimeSchema = z.object({
  suspend: z.boolean(),
  schedule: z.string().refine((value) => isValidCron(value), {
    message: "无效的 Cron 表达式，请检查格式和数值范围",
  }),
  configs: z.object({
    BATCH_DAYS: z.coerce.number().int().positive(),
    INTERACTIVE_DAYS: z.coerce.number().int().positive(),
  }),
});

const cleanLowGpuSchema = z.object({
  suspend: z.boolean(),
  schedule: z.string().refine((value) => isValidCron(value), {
    message: "无效的 Cron 表达式，请检查格式和数值范围",
  }),
  configs: z.object({
    TIME_RANGE: z.coerce.number().int().positive(),
    UTIL: z.coerce.number(),
    WAIT_TIME: z.coerce.number().int().positive(),
  }),
});

const cleanWaitingJupyterSchema = z.object({
  suspend: z.boolean(),
  schedule: z.string().refine((value) => isValidCron(value), {
    message: "无效的 Cron 表达式，请检查格式和数值范围",
  }),
  configs: z.object({
    JUPYTER_WAIT_MINUTES: z.coerce.number().int().positive(),
  }),
});

const formSchema = z.object({
  cleanLongTime: cleanLongTimeSchema,
  cleanLowGpu: cleanLowGpuSchema,
  cleanWaitingJupyter: cleanWaitingJupyterSchema,
});

type FormValues = z.infer<typeof formSchema>;

export default function CronPolicy({ className }: { className?: string }) {
  const [dryRunJobs, setDryRunJobs] = useState<string[]>([]);
  const [loading, setLoading] = useState<boolean>(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
  });

  // 加载当前策略数据，并转换为表单格式
  useEffect(() => {
    async function loadJobSchedule() {
      setLoading(true);
      try {
        const res = await apiJobScheduleAdmin();
        const detail = res.data;
        if (detail) {
          const jobs = Array.isArray(detail) ? detail : [];
          const formData: FormValues = {
            cleanLongTime:
              jobs.find((job: Job) => job.name === "clean-long-time-job") ||
              form.getValues("cleanLongTime"),
            cleanLowGpu:
              jobs.find((job: Job) => job.name === "clean-low-gpu-util-job") ||
              form.getValues("cleanLowGpu"),
            cleanWaitingJupyter:
              jobs.find((job: Job) => job.name === "clean-waiting-jupyter") ||
              form.getValues("cleanWaitingJupyter"),
          };
          form.reset(formData);
        } else {
          toast.error("获取策略失败：" + res.msg);
        }
      } catch (error) {
        toast.error("获取策略异常" + error);
      } finally {
        setLoading(false);
      }
    }
    loadJobSchedule();
  }, [form]);

  // 分别提交各个配置的更新
  const onSubmitLongTime = async () => {
    const data = form.getValues("cleanLongTime");
    try {
      const res = await apiJobScheduleChangeAdmin(data);
      if (res.code === 0) {
        toast.success("清理长时间作业更新成功");
      } else {
        toast.error("清理长时间作业更新失败：" + res.msg);
      }
    } catch (error) {
      toast.error("更新清理长时间作业策略异常" + error);
    }
  };

  const onSubmitLowGpu = async () => {
    const data = form.getValues("cleanLowGpu");
    try {
      const res = await apiJobScheduleChangeAdmin(data);
      if (res.code === 0) {
        toast.success("清理低GPU利用率作业更新成功");
      } else {
        toast.error("清理低GPU利用率作业更新失败：" + res.msg);
      }
    } catch (error) {
      toast.error("更新清理低GPU利用率策略异常" + error);
    }
  };

  const onSubmitWaitingJupyter = async () => {
    const data = form.getValues("cleanWaitingJupyter");
    try {
      const res = await apiJobScheduleChangeAdmin(data);
      if (res.code === 0) {
        toast.success("清理排队中的 jupyter 任务更新成功");
      } else {
        toast.error("清理排队中的 jupyter 任务更新失败：" + res.msg);
      }
    } catch (error) {
      toast.error("更新清理排队中的 jupyter 任务异常" + error);
    }
  };

  const runJob = async () => {
    // 模拟干跑作业逻辑
    const mockDryRunJobs = ["Job1", "Job2", "Job3"];
    setDryRunJobs(mockDryRunJobs);
  };

  const confirmJobRun = async () => {
    // 真正执行作业逻辑
    setDryRunJobs([]);
  };

  return (
    <Card className={cn("flex flex-col", className)}>
      <CardHeader>
        <CardTitle className="flex items-center gap-1.5">
          <AlarmClockIcon className="text-primary" />
          定时策略
          <TipBadge />
        </CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <Skeleton className="h-20 w-full" />
        ) : (
          <Form {...form}>
            <div className="space-y-8 p-4">
              {/* 清理长时间作业 */}
              <div className="rounded-md border p-4">
                <h3 className="mb-4 font-semibold">清理长时间作业</h3>
                <div className="flex flex-wrap gap-4">
                  <FormField
                    control={form.control}
                    name="cleanLongTime.suspend"
                    render={({ field }) => (
                      <FormItem className="flex items-center space-x-2">
                        <FormControl>
                          <Checkbox
                            checked={field.value}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                        <span>Suspend</span>
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cleanLongTime.schedule"
                    render={({ field, fieldState }) => (
                      <FormItem className="flex flex-col">
                        <label className="text-sm">执行计划</label>
                        <FormControl>
                          <Input className="mt-1 font-mono" {...field} />
                        </FormControl>
                        {fieldState.error && (
                          <p className="text-xs text-red-500">
                            {fieldState.error.message}
                          </p>
                        )}
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cleanLongTime.configs.BATCH_DAYS"
                    render={({ field, fieldState }) => (
                      <FormItem className="flex flex-col">
                        <label className="text-sm">
                          批处理任务运行天数（BATCH_DAYS）
                        </label>
                        <FormControl>
                          <Input
                            type="number"
                            className="mt-1 w-24 font-mono"
                            {...field}
                          />
                        </FormControl>
                        {fieldState.error && (
                          <p className="text-xs text-red-500">
                            {fieldState.error.message}
                          </p>
                        )}
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cleanLongTime.configs.INTERACTIVE_DAYS"
                    render={({ field, fieldState }) => (
                      <FormItem className="flex flex-col">
                        <label className="text-sm">
                          交互式任务运行天数（INTERACTIVE_DAYS）
                        </label>
                        <FormControl>
                          <Input
                            type="number"
                            className="mt-1 w-24 font-mono"
                            {...field}
                          />
                        </FormControl>
                        {fieldState.error && (
                          <p className="text-xs text-red-500">
                            {fieldState.error.message}
                          </p>
                        )}
                      </FormItem>
                    )}
                  />
                </div>
                <div className="mt-4">
                  <Button
                    variant="secondary"
                    onClick={form.handleSubmit(onSubmitLongTime)}
                  >
                    更新清理长时间作业策略
                  </Button>
                </div>
              </div>

              {/* 清理低GPU利用率作业 */}
              <div className="rounded-md border p-4">
                <h3 className="mb-4 font-semibold">清理低GPU利用率作业</h3>
                <div className="flex flex-wrap gap-4">
                  <FormField
                    control={form.control}
                    name="cleanLowGpu.suspend"
                    render={({ field }) => (
                      <FormItem className="flex items-center space-x-2">
                        <FormControl>
                          <Checkbox
                            checked={field.value}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                        <span>Suspend</span>
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cleanLowGpu.schedule"
                    render={({ field, fieldState }) => (
                      <FormItem className="flex flex-col">
                        <label className="text-sm">执行计划</label>
                        <FormControl>
                          <Input className="mt-1 font-mono" {...field} />
                        </FormControl>
                        {fieldState.error && (
                          <p className="text-xs text-red-500">
                            {fieldState.error.message}
                          </p>
                        )}
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cleanLowGpu.configs.TIME_RANGE"
                    render={({ field, fieldState }) => (
                      <FormItem className="flex flex-col">
                        <label className="text-sm">
                          时间范围（分钟，TIME_RANGE）
                        </label>
                        <FormControl>
                          <Input
                            type="number"
                            className="mt-1 w-24 font-mono"
                            {...field}
                          />
                        </FormControl>
                        {fieldState.error && (
                          <p className="text-xs text-red-500">
                            {fieldState.error.message}
                          </p>
                        )}
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cleanLowGpu.configs.UTIL"
                    render={({ field, fieldState }) => (
                      <FormItem className="flex flex-col">
                        <label className="text-sm">利用率（UTIL）</label>
                        <FormControl>
                          <Input
                            type="number"
                            step="0.1"
                            className="mt-1 w-24 font-mono"
                            {...field}
                          />
                        </FormControl>
                        {fieldState.error && (
                          <p className="text-xs text-red-500">
                            {fieldState.error.message}
                          </p>
                        )}
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cleanLowGpu.configs.WAIT_TIME"
                    render={({ field, fieldState }) => (
                      <FormItem className="flex flex-col">
                        <label className="text-sm">
                          等待时间（WAIT_TIME，分钟）
                        </label>
                        <FormControl>
                          <Input
                            type="number"
                            className="mt-1 w-24 font-mono"
                            {...field}
                          />
                        </FormControl>
                        {fieldState.error && (
                          <p className="text-xs text-red-500">
                            {fieldState.error.message}
                          </p>
                        )}
                      </FormItem>
                    )}
                  />
                </div>
                <div className="mt-4">
                  <Button
                    variant="secondary"
                    onClick={form.handleSubmit(onSubmitLowGpu)}
                  >
                    更新清理低GPU利用率策略
                  </Button>
                </div>
              </div>

              {/* 清理排队中的 jupyter 任务 */}
              <div className="rounded-md border p-4">
                <h3 className="mb-4 font-semibold">
                  清理排队中的 jupyter 任务
                </h3>
                <div className="flex flex-wrap gap-4">
                  <FormField
                    control={form.control}
                    name="cleanWaitingJupyter.suspend"
                    render={({ field }) => (
                      <FormItem className="flex items-center space-x-2">
                        <FormControl>
                          <Checkbox
                            checked={field.value}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                        <span>Suspend</span>
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cleanWaitingJupyter.schedule"
                    render={({ field, fieldState }) => (
                      <FormItem className="flex flex-col">
                        <label className="text-sm">执行计划</label>
                        <FormControl>
                          <Input className="mt-1 font-mono" {...field} />
                        </FormControl>
                        {fieldState.error && (
                          <p className="text-xs text-red-500">
                            {fieldState.error.message}
                          </p>
                        )}
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cleanWaitingJupyter.configs.JUPYTER_WAIT_MINUTES"
                    render={({ field, fieldState }) => (
                      <FormItem className="flex flex-col">
                        <label className="text-sm">
                          等待时间（JUPYTER_WAIT_MINUTES，分钟）
                        </label>
                        <FormControl>
                          <Input
                            type="number"
                            className="mt-1 w-24 font-mono"
                            {...field}
                          />
                        </FormControl>
                        {fieldState.error && (
                          <p className="text-xs text-red-500">
                            {fieldState.error.message}
                          </p>
                        )}
                      </FormItem>
                    )}
                  />
                </div>
                <div className="mt-4">
                  <Button
                    variant="secondary"
                    onClick={form.handleSubmit(onSubmitWaitingJupyter)}
                  >
                    更新清理排队中的 jupyter 策略
                  </Button>
                </div>
              </div>
            </div>
          </Form>
        )}
      </CardContent>
      <CardFooter className="flex flex-wrap items-center gap-4 p-4">
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button onClick={runJob} variant="destructive">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
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
      </CardFooter>
    </Card>
  );
}
