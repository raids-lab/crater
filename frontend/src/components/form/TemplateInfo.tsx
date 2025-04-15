import React from "react";
import { UIStateUpdater, useTemplateLoader } from "@/hooks/useTemplateLoader";
import { FieldValues, UseFormReturn } from "react-hook-form";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
} from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import { Badge } from "@/components/ui/badge";
import { MarkdownRenderer } from "./markdown-renderer";

interface TemplateInfoProps<T extends FieldValues> {
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

export function TemplateInfo<T extends FieldValues>({
  form,
  metadata,
  uiStateUpdaters = [],
  onSuccess,
  dataProcessor,
}: TemplateInfoProps<T>) {
  // 使用 hook 获取模板信息
  const { fromJob, fromTemplate, templateData } = useTemplateLoader({
    form,
    metadata,
    uiStateUpdaters,
    onSuccess,
    dataProcessor,
  });

  // 缓存Markdown内容以避免不必要的重渲染
  const markdownContent = React.useMemo(() => {
    return templateData?.document || "";
  }, [templateData?.document]);

  // 如果没有模板，不渲染任何内容
  if (!fromJob && !fromTemplate) {
    return null;
  }

  return (
    <Card className="mb-4">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center">
            {fromJob ? "作业模板" : "预设模板"}
            <Badge variant="outline" className="ml-2">
              {fromJob || templateData?.name}
            </Badge>
          </CardTitle>

          {templateData?.userInfo.nickname && (
            <Badge variant="secondary">
              作者: {templateData.userInfo.nickname}
            </Badge>
          )}
        </div>

        <CardDescription>
          {fromJob
            ? `已加载来自作业 ${fromJob} 的模板配置`
            : templateData?.describe}
        </CardDescription>
      </CardHeader>

      {markdownContent && (
        <CardContent className="pt-0">
          <div className="">
            <MarkdownRenderer>{markdownContent}</MarkdownRenderer>
          </div>
        </CardContent>
      )}
    </Card>
  );
}
