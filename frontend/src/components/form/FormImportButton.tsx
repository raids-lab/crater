// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { importFromJsonFile } from "@/utils/form";
import { showErrorToast } from "@/utils/toast";
import { CircleArrowDown } from "lucide-react";
import { FieldValues, UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { MetadataFormType } from "./types";
import TooltipButton from "../custom/TooltipButton";

interface ImportButtonProps<T extends FieldValues> {
  metadata: MetadataFormType;
  form?: UseFormReturn<T>;
  beforeImport?: (data: T) => void;
  dataProcessor?: (data: T) => T;
  afterImport?: (data: T) => void;
  buttonText?: string;
  className?: string;
}

function FormImportButton<T extends FieldValues>({
  metadata,
  beforeImport,
  dataProcessor,
  afterImport,
  form,
  className,
  buttonText,
}: ImportButtonProps<T>) {
  const { t } = useTranslation();

  return (
    <TooltipButton
      variant="outline"
      type="button"
      className={cn("relative cursor-pointer", className)}
      tooltipContent={t("formImportButton.tooltip")}
    >
      <Input
        onChange={(e) => {
          importFromJsonFile<T>(
            metadata.version,
            metadata.type,
            e.target.files?.[0],
          )
            .then((data) => {
              beforeImport?.(data);
              // Apply optional data processing
              if (dataProcessor) {
                data = dataProcessor(data);
              }
              form?.reset(data);
              afterImport?.(data);
              toast.info(t("formImportButton.successToast"));
            })
            .catch((error) => {
              showErrorToast(error);
            });
        }}
        type="file"
        className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
      />
      <CircleArrowDown className="size-4" />
      {buttonText || t("formImportButton.buttonText")}
    </TooltipButton>
  );
}

export default FormImportButton;
