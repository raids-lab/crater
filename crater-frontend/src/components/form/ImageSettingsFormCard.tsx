// i18n-processed-v1.1.0
// Modified code
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
import { FieldPath, FieldValues, UseFormReturn } from "react-hook-form";
import { Separator } from "@/components/ui/separator";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import TipBadge from "@/components/badge/TipBadge";

export function ImageSettingsFormCard<T extends FieldValues>({
  form,
  imageNamePath,
  imageTagPath,
  description = "输入用户自定义的镜像名和镜像标签，若为空，则由系统自动生成",
  className,
}: ImageSettingsFormCardProps<T>) {
  const { t } = useTranslation();

  return (
    <Accordion
      type="single"
      collapsible
      className={`w-full rounded-lg border ${className}`}
    >
      <AccordionItem value="image-settings" className="border-none">
        <AccordionTrigger className="px-4 py-3 hover:no-underline">
          <div className="flex flex-row items-center gap-1.5">
            {t("imageSettingsCard")}
            <TipBadge title={t("imageSettingsForm.tipBadgeTitle")} />
          </div>
        </AccordionTrigger>
        <AccordionContent>
          <Separator />
          <div className="flex items-start gap-4 px-4 pt-4">
            <div className="flex-1">
              <FormField
                control={form.control}
                name={imageNamePath}
                render={({ field }) => (
                  <FormItem className="flex h-full flex-col">
                    <FormLabel>{t("imageSettingsForm.nameLabel")}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder={t("imageSettingsForm.namePlaceholder")}
                      />
                    </FormControl>
                    <FormMessage className="min-h-[20px] leading-none" />
                  </FormItem>
                )}
              />
            </div>
            <div className="flex-1">
              <FormField
                control={form.control}
                name={imageTagPath}
                render={({ field }) => (
                  <FormItem className="flex h-full flex-col">
                    <FormLabel>{t("imageSettingsForm.tagLabel")}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder={t("imageSettingsForm.tagPlaceholder")}
                      />
                    </FormControl>
                    <FormMessage className="min-h-[20px] leading-none" />
                  </FormItem>
                )}
              />
            </div>
          </div>
          <FormDescription className="mt-2 px-4 pb-4">
            {t("imageSettingsForm.description", { description })}
          </FormDescription>
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  );
}

interface ImageSettingsFormCardProps<T extends FieldValues> {
  form: UseFormReturn<T>;
  imageNamePath: FieldPath<T>;
  imageTagPath: FieldPath<T>;
  description?: string;
  className?: string;
}
