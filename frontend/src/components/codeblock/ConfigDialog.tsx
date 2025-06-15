// Reference: https://github.com/kubesphere/console/blob/master/packages/shared/src/stores/pod.ts#L187
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";
import { Card } from "@/components/ui/card";
import { FetchSheet } from "./Dialog";
import useResizeObserver from "use-resize-observer";
import { ScrollArea, ScrollBar } from "../ui/scroll-area";
import { CopyButton } from "../button/copy-button";
import { HighlightedPre } from "../form/markdown-renderer";
import { cn } from "@/lib/utils";

export interface PodNamespacedName {
  namespace: string;
  name: string;
}

export function CodeContent({
  data,
  language,
  className,
  moreActions,
}: {
  data: string;
  language?: string;
  className?: string;
  moreActions?: React.ReactNode;
}) {
  const { ref: refRoot, width, height } = useResizeObserver();

  return (
    <Card
      className={cn(
        "text-muted-foreground bg-muted/30 relative h-full overflow-hidden p-1 dark:border",
        className,
      )}
      ref={refRoot}
    >
      <ScrollArea style={{ width, height }}>
        <HighlightedPre
          className="p-2 text-sm"
          withLineNumbers
          language={language ?? ""}
        >
          {data ?? ""}
        </HighlightedPre>
        <ScrollBar orientation="horizontal" />
      </ScrollArea>
      <div className="absolute top-5 right-5 flex h-8 flex-row items-center gap-2">
        {moreActions}
        <CopyButton
          className="size-8"
          content={data ?? ""}
          variant="outline"
          size="icon"
        />
      </div>
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
