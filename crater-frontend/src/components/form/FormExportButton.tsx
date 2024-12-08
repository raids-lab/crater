import { Button } from "@/components/ui/button";
import { exportToJsonFile } from "@/utils/form";
import { showErrorToast } from "@/utils/toast";
import { CircleArrowUp } from "lucide-react";
import { FieldValues, UseFormReturn } from "react-hook-form";
import { MetadataFormType } from "./types";

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
    <Button
      variant="outline"
      type="button"
      className={className}
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
      <CircleArrowUp className="h-4 w-4" />
      {buttonText}
    </Button>
  );
}

export default FormExportButton;
