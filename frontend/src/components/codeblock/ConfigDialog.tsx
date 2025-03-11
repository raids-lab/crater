// Reference: https://github.com/kubesphere/console/blob/master/packages/shared/src/stores/pod.ts#L187
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";
import { Card } from "@/components/ui/card";
import BaseCodeBlock from "./BaseCodeBlock";
import { CopyCheckIcon, CopyIcon } from "lucide-react";
import { toast } from "sonner";
import { useCopyToClipboard } from "usehooks-ts";
import { FetchSheet } from "./Dialog";
import useResizeObserver from "use-resize-observer";
import { ScrollArea } from "../ui/scroll-area";
import TooltipButton from "../custom/TooltipButton";

export interface PodNamespacedName {
  namespace: string;
  name: string;
}

export function CodeContent({
  data: yaml,
  language,
}: {
  data: string;
  language?: string;
}) {
  const { ref: refRoot, width, height } = useResizeObserver();
  const [copiedText, copy] = useCopyToClipboard();

  const copyCode = () => {
    // Logic to copy `code`
    copy(yaml ?? "")
      .then(() => {
        toast.success("已复制到剪贴板");
      })
      .catch(() => {
        toast.error("复制失败");
      });
  };

  return (
    <Card
      className="text-muted-foreground dark:bg-muted/30 relative h-[calc(100vh-_304px)] overflow-hidden bg-slate-900 p-1 dark:border"
      ref={refRoot}
    >
      <ScrollArea style={{ width, height }}>
        <BaseCodeBlock code={yaml ?? ""} language={language ?? ""} />
      </ScrollArea>
      <TooltipButton
        tooltipContent="复制"
        className="absolute top-5 right-5 h-8 w-8"
        onClick={copyCode}
        variant="outline"
        size="icon"
      >
        {copiedText ? (
          <CopyCheckIcon className="size-4" />
        ) : (
          <CopyIcon className="size-4" />
        )}
      </TooltipButton>
    </Card>
  );
}

interface ConfigContentProps {
  jobName: string;
  language?: string;
  getConfig: (
    name: string,
  ) => Promise<AxiosResponse<IResponse<string>, unknown>>;
}

type ConfigDialogProps = React.HTMLAttributes<HTMLDivElement> &
  ConfigContentProps;

export function ConfigDialog({
  children,
  jobName,
  language,
  getConfig,
}: ConfigDialogProps) {
  return (
    <FetchSheet
      trigger={children}
      name={jobName}
      type="yaml"
      fetchData={getConfig}
      renderData={(data) => <CodeContent data={data} language={language} />}
    />
  );
}
