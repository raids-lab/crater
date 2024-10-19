// Reference: https://github.com/kubesphere/console/blob/master/packages/shared/src/stores/pod.ts#L187
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useQuery } from "@tanstack/react-query";
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

export interface PodNamespacedName {
  namespace: string;
  name: string;
}

function Content({
  jobName,
  language = "yaml",
  getConfig: getYaml,
}: ConfigContentProps) {
  const { ref: refRoot, width, height } = useResizeObserver();
  const [copiedText, copy] = useCopyToClipboard();

  const { data: yaml } = useQuery({
    queryKey: ["yaml", "job", jobName],
    queryFn: () => getYaml(jobName),
    select: (res) => res.data.data,
    enabled: !!jobName,
  });

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
    <>
      <DialogHeader>
        <DialogTitle className="font-semibold">
          <span className="font-mono">{jobName}</span>
        </DialogTitle>
      </DialogHeader>
      <Card
        className="relative h-[calc(100vh_-292px)] w-[calc(100vw_-252px)] overflow-hidden bg-slate-900 p-1 text-muted-foreground dark:border"
        ref={refRoot}
      >
        <ScrollArea style={{ width, height }}>
          <BaseCodeBlock code={yaml ?? ""} language={language} />
        </ScrollArea>
        <Button
          className="absolute right-5 top-5 h-8 w-8"
          onClick={copyCode}
          variant="outline"
          size="icon"
        >
          {copiedText ? (
            <CopyCheckIcon className="h-4 w-4" />
          ) : (
            <CopyIcon className="h-4 w-4" />
          )}
        </Button>
      </Card>
    </>
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
    <Dialog>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent
        className="h-[calc(100vh_-200px)] w-[calc(100vw_-200px)] max-w-full gap-5"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <Content jobName={jobName} getConfig={getConfig} language={language} />
      </DialogContent>
    </Dialog>
  );
}
