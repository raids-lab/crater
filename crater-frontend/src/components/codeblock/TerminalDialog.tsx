import { ScrollArea } from "@/components/ui/scroll-area";
import useResizeObserver from "use-resize-observer";
import { Card } from "../ui/card";
import { ContainerInfo } from "@/services/api/tool";
import { PodContainerDialog, PodNamespacedName } from "./PodContainerDialog";

function TerminalCard({
  namespacedName,
  selectedContainer,
}: {
  namespacedName: PodNamespacedName;
  selectedContainer: ContainerInfo;
}) {
  const { ref: refRoot, width, height } = useResizeObserver();

  if (!selectedContainer.state.running) {
    return <></>;
  }

  return (
    <Card
      className="relative h-full overflow-hidden rounded-md bg-slate-900 p-1 text-white dark:border dark:bg-muted/30 md:col-span-2 xl:col-span-3"
      ref={refRoot}
    >
      <ScrollArea style={{ width, height }}>
        <pre className="whitespace-pre-wrap break-words px-3 py-3 text-sm text-cyan-200 dark:text-muted-foreground">
          Connecting to terminal of {namespacedName.namespace}/
          {namespacedName.name}/{selectedContainer.name}...
        </pre>
      </ScrollArea>
    </Card>
  );
}

export default function TerminalDialog({
  namespacedName,
  setNamespacedName,
}: {
  namespacedName?: PodNamespacedName;
  setNamespacedName: (namespacedName: PodNamespacedName | undefined) => void;
}) {
  return (
    <PodContainerDialog
      namespacedName={namespacedName}
      setNamespacedName={setNamespacedName}
      ActionComponent={TerminalCard}
    />
  );
}
