import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { FieldPath, FieldValues, UseFormReturn } from "react-hook-form";
import AccordionCard from "./AccordionCard";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { CircleHelpIcon } from "lucide-react";

export const OtherCard = "其他选项";

interface OtherOptionsFormCardProps<T extends FieldValues> {
  form: UseFormReturn<T>;
  alertEnabledPath: FieldPath<T>;
  nodeSelectorEnablePath: FieldPath<T>;
  nodeSelectorNodeNamePath: FieldPath<T>;
  value: string | undefined;
  setValue: (value: string) => void;
}

export function OtherOptionsFormCard<T extends FieldValues>({
  form,
  alertEnabledPath,
  nodeSelectorEnablePath,
  nodeSelectorNodeNamePath,
  value,
  setValue,
}: OtherOptionsFormCardProps<T>) {
  const nodeSelectorEnabled = form.watch(nodeSelectorEnablePath);

  return (
    <AccordionCard cardTitle={OtherCard} value={value} setValue={setValue}>
      <div className="mt-3 space-y-3">
        <FormField
          control={form.control}
          name={alertEnabledPath}
          render={({ field }) => (
            <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
              <FormLabel className="font-normal">
                接收状态通知
                <TooltipProvider delayDuration={100}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>通过邮件接收作业状态通知：</p>
                      <p>1. 作业开始运行通知（等待超过5分钟）</p>
                      <p>2. 作业运行成功通知</p>
                      <p>3. 作业运行失败通知</p>
                      <p>4. 作业低利用率/长运行时间即将释放通知</p>
                      <p>5. 作业低利用率/长运行时间已释放通知</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </FormLabel>
              <FormControl>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
              </FormControl>
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name={nodeSelectorEnablePath}
          render={({ field }) => (
            <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
              <FormLabel className="font-normal">
                指定工作节点
                <TooltipProvider delayDuration={100}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                    </TooltipTrigger>
                    <TooltipContent>
                      用于调试，或对特定节点进行性能测试场景
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </FormLabel>
              <FormControl>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
              </FormControl>
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name={nodeSelectorNodeNamePath}
          render={({ field }) => (
            <FormItem
              className={cn({
                hidden: !nodeSelectorEnabled,
              })}
            >
              <FormControl>
                <Input {...field} className="font-mono" />
              </FormControl>
              <FormDescription>
                节点名称（可通过空闲资源查询获取）
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>
    </AccordionCard>
  );
}
