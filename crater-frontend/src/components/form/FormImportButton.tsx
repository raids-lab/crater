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
  afterImport?: (data: T) => void;
  buttonText?: string;
  className?: string;
}

function FormImportButton<T extends FieldValues>({
  metadata,
  beforeImport,
  afterImport,
  form,
  className,
  buttonText = "导入配置",
}: ImportButtonProps<T>) {
  return (
    <TooltipButton
      variant="outline"
      type="button"
      className={cn("relative cursor-pointer", className)}
      tooltipContent="从本地文件导入作业配置"
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
              form?.reset(data);
              afterImport?.(data);
              toast.info("导入配置成功");
            })
            .catch((error) => {
              showErrorToast(error);
            });
        }}
        type="file"
        className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
      />
      <CircleArrowDown className="size-4" />
      {buttonText}
    </TooltipButton>
  );
}

export default FormImportButton;
