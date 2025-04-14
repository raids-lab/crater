// Reference: https://github.com/kubesphere/console/blob/master/packages/shared/src/stores/pod.ts#L187
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";
import { Card } from "@/components/ui/card";
import BaseCodeBlock from "./BaseCodeBlock";
import { FetchSheet } from "./Dialog";
import useResizeObserver from "use-resize-observer";
import { ScrollArea } from "../ui/scroll-area";
import { CopyButton } from "../button/copy-button";

export interface PodNamespacedName {
  namespace: string;
  name: string;
}

export function CodeContent({
  data,
  language,
}: {
  data: string;
  language?: string;
}) {
  const { ref: refRoot, width, height } = useResizeObserver();

  return (
    <Card
      className="text-muted-foreground bg-muted/30 relative h-full overflow-hidden p-1 dark:border"
      ref={refRoot}
    >
      <ScrollArea style={{ width, height }}>
        <BaseCodeBlock code={data ?? ""} language={language ?? ""} />
      </ScrollArea>
      <CopyButton
        className="absolute top-5 right-5 h-8 w-8"
        content={data ?? ""}
        variant="outline"
        size="icon"
      />
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
