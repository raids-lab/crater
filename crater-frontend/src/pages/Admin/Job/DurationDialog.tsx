"use client";

import { useEffect, useState } from "react";
import { addHours, format } from "date-fns";
import { zhCN } from "date-fns/locale";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { CalendarClock, InfoIcon, Lock, UnlockIcon } from "lucide-react";
import { IJobInfo, apiJobLock, apiJobUnlock } from "@/services/api/vcjob";
import { toast } from "sonner";
import { useMutation } from "@tanstack/react-query";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";

// 定义表单验证模式
const formSchema = z.object({
  isPermanent: z.boolean().default(false),
  days: z.coerce.number().min(0, "天数不能为负数").default(0),
  hours: z.coerce
    .number()
    .min(0, "小时数不能为负数")
    .max(23, "小时数不能超过23")
    .default(0),
});

type FormValues = z.infer<typeof formSchema>;

interface DurationDialogProps {
  jobs: IJobInfo[];
  open: boolean;
  setOpen: (open: boolean) => void;
  onSuccess?: () => void;
}

export function DurationDialog({
  jobs,
  open,
  setOpen,
  onSuccess,
}: DurationDialogProps) {
  const [expirationDate, setExpirationDate] = useState<Date | null>(null);
  const allLocked = jobs.length > 0 && jobs.every((job) => job.locked);
  const allUnlocked = jobs.length > 0 && jobs.every((job) => !job.locked);
  const mixedState = !allLocked && !allUnlocked;

  // 初始化表单
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      isPermanent: false,
      days: 0,
      hours: 0,
    },
  });

  // 使用 React Query 的 useMutation 进行锁定操作
  const lockMutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const { isPermanent, days, hours } = values;
      const promises = jobs.map((job) => {
        const payload = {
          name: job.jobName,
          isPermanent,
          days: days || 0,
          hours: hours || 0,
          minutes: 0, // 始终为 0
        };
        return apiJobLock(payload);
      });
      return Promise.all(promises);
    },
    onSuccess: () => {
      toast.success(`成功锁定 ${jobs.length} 个作业`);
      setOpen(false);
      onSuccess?.();
    },
    onError: () => {
      toast.error("锁定操作失败");
    },
  });

  // 使用 React Query 的 useMutation 进行解锁操作
  const unlockMutation = useMutation({
    mutationFn: async () => {
      const promises = jobs.map((job) => apiJobUnlock(job.jobName));
      return Promise.all(promises);
    },
    onSuccess: () => {
      toast.success(`成功解锁 ${jobs.length} 个作业`);
      setOpen(false);
      onSuccess?.();
    },
    onError: () => {
      toast.error("解锁操作失败");
    },
  });

  // 重置表单当对话框打开时
  useEffect(() => {
    if (open) {
      form.reset({
        isPermanent: false,
        days: 0,
        hours: 0,
      });
      setExpirationDate(null);
    }
  }, [open, form]);

  // 计算到期时间的函数
  const calculateExpirationDate = (values: FormValues) => {
    const { days, hours, isPermanent } = values;

    if (isPermanent) {
      setExpirationDate(null);
      return;
    }

    // 只有当至少有一个值大于0时才计算
    if (days > 0 || hours > 0) {
      // 创建一个新的日期对象
      const now = new Date();
      let result = new Date(now);

      // 添加时间
      if (days > 0) result = addHours(result, days * 24);
      if (hours > 0) result = addHours(result, hours);

      setExpirationDate(result);
    } else {
      setExpirationDate(null);
    }
  };

  // 处理字段变化事件
  const handleFieldChange = () => {
    setTimeout(() => {
      const values = form.getValues();
      calculateExpirationDate(values);
    }, 0);
  };

  // 提交表单
  async function onSubmit(values: FormValues) {
    const { isPermanent, days, hours } = values;

    // 验证输入
    if (!isPermanent && !(days > 0 || hours > 0)) {
      toast.error("请设置锁定时长或选择永久锁定");
      return;
    }

    lockMutation.mutate(values);
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{allLocked ? "解锁作业" : "锁定作业"}</DialogTitle>
          <DialogDescription>
            {allLocked
              ? `解锁 ${jobs.length} 个作业，解锁后作业将会根据策略自动清理。`
              : `设置 ${jobs.length} 个作业的锁定时长，锁定后作业不会被自动清理。`}
          </DialogDescription>
        </DialogHeader>

        {mixedState ? (
          <div className="py-4">
            <Card className="border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950">
              <CardContent className="flex items-start gap-2 pt-6">
                <InfoIcon className="mt-0.5 h-5 w-5 flex-shrink-0 text-amber-500" />
                <div>
                  <p className="font-medium">选中的作业状态不一致</p>
                  <p className="text-muted-foreground text-sm">
                    请选择状态一致的作业（全部已锁定或全部未锁定）进行批量操作。
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        ) : allLocked ? (
          <Button
            onClick={() => unlockMutation.mutate()}
            className="w-full"
            variant="outline"
            disabled={unlockMutation.isPending}
          >
            <UnlockIcon className="mr-2 h-4 w-4" />
            {unlockMutation.isPending
              ? "解锁中..."
              : `解锁 ${jobs.length} 个作业`}
          </Button>
        ) : (
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <FormField
                control={form.control}
                name="isPermanent"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-start space-y-0 space-x-3 rounded-md border p-4">
                    <FormControl>
                      <Checkbox
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        onBlur={field.onBlur}
                        onClick={() => {
                          setTimeout(handleFieldChange, 0);
                        }}
                      />
                    </FormControl>
                    <div className="space-y-1 leading-none">
                      <FormLabel className="text-base">永久锁定</FormLabel>
                      <p className="text-muted-foreground text-sm">
                        作业将被永久锁定，不会被自动清理策略删除。
                      </p>
                    </div>
                  </FormItem>
                )}
              />

              {!form.watch("isPermanent") && (
                <>
                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="days"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>天数</FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              min="0"
                              placeholder="0"
                              {...field}
                              onChange={(e) => {
                                field.onChange(e);
                                handleFieldChange();
                              }}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="hours"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>小时</FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              min="0"
                              max="23"
                              placeholder="0"
                              {...field}
                              onChange={(e) => {
                                field.onChange(e);
                                handleFieldChange();
                              }}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  {expirationDate && (
                    <Card className="border-dashed">
                      <CardContent>
                        <div className="text-muted-foreground flex items-center gap-2">
                          <CalendarClock className="h-4 w-4" />
                          <span>到期时间预览：</span>
                        </div>
                        <div className="mt-2">
                          <p className="text-lg font-medium">
                            {format(expirationDate, "yyyy年MM月dd日 HH:mm", {
                              locale: zhCN,
                            })}
                          </p>
                          <p className="text-muted-foreground mt-1 text-sm">
                            当前时间:{" "}
                            {format(new Date(), "yyyy年MM月dd日 HH:mm", {
                              locale: zhCN,
                            })}
                          </p>
                        </div>
                      </CardContent>
                    </Card>
                  )}
                </>
              )}

              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setOpen(false)}
                  disabled={lockMutation.isPending}
                >
                  取消
                </Button>
                <Button type="submit" disabled={lockMutation.isPending}>
                  <Lock className="mr-2 h-4 w-4" />
                  {lockMutation.isPending ? "锁定中..." : "锁定作业"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        )}
      </DialogContent>
    </Dialog>
  );
}
