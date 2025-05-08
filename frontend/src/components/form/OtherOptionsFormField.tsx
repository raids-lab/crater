// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
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

export function getOtherCardTitle(t: (key: string) => string) {
  return t("otherOptionsFormCard.accordionTitle");
}

interface OtherOptionsFormCardProps<T extends FieldValues> {
  form: UseFormReturn<T>;
  alertEnabledPath: FieldPath<T>;
  nodeSelectorEnablePath: FieldPath<T>;
  nodeSelectorNodeNamePath: FieldPath<T>;
  open: boolean;
  setOpen: (open: boolean) => void;
}

export function OtherOptionsFormCard<T extends FieldValues>({
  form,
  alertEnabledPath,
  nodeSelectorEnablePath,
  nodeSelectorNodeNamePath,
  open,
  setOpen,
}: OtherOptionsFormCardProps<T>) {
  const { t } = useTranslation();
  const nodeSelectorEnabled = form.watch(nodeSelectorEnablePath);

  return (
    <AccordionCard
      cardTitle={t("otherOptionsFormCard.accordionTitle")}
      open={open}
      setOpen={setOpen}
    >
      <div className="mt-3 space-y-3">
        <FormField
          control={form.control}
          name={alertEnabledPath}
          render={({ field }) => (
            <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
              <FormLabel className="font-normal">
                {t("otherOptionsFormCard.receiveStatusNotifications")}
                <TooltipProvider delayDuration={100}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                    </TooltipTrigger>
                    <TooltipContent>
                      <p className="mb-0.5 font-semibold">
                        {t(
                          "otherOptionsFormCard.tooltip.receiveEmailNotifications",
                        )}
                      </p>
                      <p>{t("otherOptionsFormCard.tooltip.notification1")}</p>
                      <p>{t("otherOptionsFormCard.tooltip.notification2")}</p>
                      <p>{t("otherOptionsFormCard.tooltip.notification3")}</p>
                      <p>{t("otherOptionsFormCard.tooltip.notification4")}</p>
                      <p>{t("otherOptionsFormCard.tooltip.notification5")}</p>
                      <p>{t("otherOptionsFormCard.tooltip.emailSupport")}</p>
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
        <div className="space-y-1.5">
          <FormField
            control={form.control}
            name={nodeSelectorEnablePath}
            render={({ field }) => (
              <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
                <FormLabel className="font-normal">
                  {t("otherOptionsFormCard.specifyWorkNode")}
                  <TooltipProvider delayDuration={100}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                      </TooltipTrigger>
                      <TooltipContent>
                        {t(
                          "otherOptionsFormCard.tooltip.debugPerformanceTesting",
                        )}
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
                  {t("otherOptionsFormCard.nodeNameDescription")}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      </div>
    </AccordionCard>
  );
}
