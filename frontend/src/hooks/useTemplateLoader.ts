import { useEffect } from "react";
import { FieldValues, UseFormReturn } from "react-hook-form";
import { useSearchParams } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import { apiJobTemplate } from "@/services/api/vcjob";
import { importFromJsonString } from "@/utils/form";
import { toast } from "sonner";
import { showErrorToast } from "@/utils/toast";

interface UIStateUpdater<T> {
  /** Condition to determine if this state should be updated */
  condition: (data: T) => boolean;
  /** State setter function */
  setter: (value: string) => void;
  /** Value to set if condition is true */
  value: string;
}

interface TemplateLoaderOptions<T extends FieldValues> {
  /** The form object to populate */
  form: UseFormReturn<T>;
  /** Metadata configuration for import/export */
  metadata: { version: string; type: string };
  /** Optional UI state updaters for accordions, tabs, etc. */
  uiStateUpdaters?: UIStateUpdater<T>[];
  /** Optional callback when template loaded successfully */
  onSuccess?: (data: T) => void;
  /** Optional additional data processing */
  dataProcessor?: (data: T) => T;
}

/**
 * Custom hook for loading job templates from URL params
 * @example
 * const { fromJob } = useTemplateLoader({
 *   form,
 *   metadata: MetadataFormJupyter,
 *   uiStateUpdaters: [
 *     { condition: data => data.envs.length > 0, setter: setEnvOpen, value: "环境变量" },
 *     { condition: data => data.nodeSelector.enable, setter: setOtherOpen, value: "其他设置" },
 *   ]
 * });
 */
export function useTemplateLoader<T extends FieldValues>({
  form,
  metadata,
  uiStateUpdaters = [],
  onSuccess,
  dataProcessor,
}: TemplateLoaderOptions<T>) {
  const [searchParams] = useSearchParams();
  const fromJob = searchParams.get("fromJob");

  const { mutate: loadJobTemplate } = useMutation({
    mutationFn: (jobName: string) => apiJobTemplate(jobName),
    onSuccess: (response) => {
      try {
        // Import the template data
        let jobInfo = importFromJsonString<T>(metadata, response.data.data);

        // Apply optional data processing
        if (dataProcessor) {
          jobInfo = dataProcessor(jobInfo);
        }

        // Reset the form with the loaded data
        form.reset(jobInfo);

        // Update UI states based on the loaded data
        uiStateUpdaters.forEach(({ condition, setter, value }) => {
          if (condition(jobInfo)) {
            setter(value);
          }
        });

        // Call the success callback
        if (onSuccess) {
          onSuccess(jobInfo);
        }

        toast.success(`已加载作业 ${fromJob} 的模板配置`);
      } catch (error) {
        showErrorToast(error);
      }
    },
    onError: () => {
      toast.error("获取作业模板失败");
    },
  });

  // Load template when fromJob changes
  useEffect(() => {
    if (fromJob) {
      loadJobTemplate(fromJob);
    }
  }, [loadJobTemplate, fromJob]);

  return { fromJob };
}
