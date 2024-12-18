// Reference: https://github.com/kubesphere/console/blob/master/packages/shared/src/stores/pod.ts#L187
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Card } from "@/components/ui/card";
import BaseCodeBlock from "./BaseCodeBlock";
import { Button } from "../ui/button";
import { CopyCheckIcon, CopyIcon } from "lucide-react";
import { toast } from "sonner";
import useResizeObserver from "use-resize-observer";
import { useCopyToClipboard } from "usehooks-ts";
import { CodeDialog } from "./Dialog";

export interface PodNamespacedName {
  namespace: string;
  name: string;
}

function Content({
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
      className="relative h-[calc(100vh_-292px)] w-[calc(100vw_-252px)] overflow-hidden bg-slate-900 p-1 text-muted-foreground dark:border"
      ref={refRoot}
    >
      <ScrollArea style={{ width, height }}>
        <BaseCodeBlock code={yaml ?? ""} language={language ?? ""} />
      </ScrollArea>
      <Button
        className="absolute right-5 top-5 h-8 w-8"
        onClick={copyCode}
        variant="outline"
        size="icon"
      >
        {copiedText ? (
          <CopyCheckIcon className="size-4" />
        ) : (
          <CopyIcon className="size-4" />
        )}
      </Button>
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
    <CodeDialog
      trigger={children}
      name={jobName}
      type="yaml"
      fetchData={getConfig}
      renderData={(data) => <Content data={data} language={language} />}
    />
  );
}
