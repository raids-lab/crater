import { exportToJsonFile } from "@/utils/form";
import { showErrorToast } from "@/utils/toast";
import { CircleArrowUp } from "lucide-react";
import { FieldValues, UseFormReturn } from "react-hook-form";
import { MetadataFormType } from "./types";
import TooltipButton from "../custom/TooltipButton";

interface ImportButtonProps<T extends FieldValues> {
  metadata: MetadataFormType;
  form?: UseFormReturn<T>;
  buttonText?: string;
  className?: string;
}

function FormExportButton<T extends FieldValues>({
  metadata,
  form,
  className,
  buttonText = "导出配置",
}: ImportButtonProps<T>) {
  const currentValues = form?.getValues();
  return (
    <TooltipButton
      variant="outline"
      type="button"
      className={className}
      tooltipContent="将当前配置导出到本地文件"
      onClick={() => {
        form
          ?.trigger()
          .then((isValid) => {
            if (!isValid) {
              return;
            }
            // 导出为小写 formType + 当前日期 MMDD + .json
            // 例如：job-0910.json
            const fileName = `${metadata.type.toLowerCase()}_${new Date().toLocaleDateString(
              "en-US",
              {
                month: "2-digit",
                day: "2-digit",
              },
            )}.json`;
            exportToJsonFile(
              {
                version: metadata.version,
                type: metadata.type,
                data: currentValues,
              },
              fileName,
            );
          })
          .catch((error) => {
            showErrorToast(error);
          });
      }}
    >
      <CircleArrowUp className="size-4" />
      {buttonText}
    </TooltipButton>
  );
}

export default FormExportButton;
