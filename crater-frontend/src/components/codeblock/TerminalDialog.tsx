import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import useResizeObserver from "use-resize-observer";
import { Card } from "../ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

function ContainerSelect() {
  return (
    <Select>
      <SelectTrigger className="w-full">
        <SelectValue placeholder="Select a worker" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="worker1">
          <div className="flex items-center space-x-2">
            <div className="h-8 w-8 rounded-full bg-blue-500"></div>
            <div>
              <h3 className="font-semibold">worker</h3>
              <p className="text-sm text-gray-500">
                镜像：registry.k8s.io/nfd/node-fe...
              </p>
            </div>
          </div>
        </SelectItem>
        {/* Add more SelectItem components for additional workers */}
      </SelectContent>
    </Select>
  );
}

function Content({ podName }: { podName: string }) {
  const { ref: refRoot, width, height } = useResizeObserver();

  // const [isOpen, setIsOpen] = useState(true);
  const [terminalOutput, setTerminalOutput] = useState<string[]>([
    "Connecting...",
  ]);

  useEffect(() => {
    // Simulate terminal connection and output
    const timer = setTimeout(() => {
      setTerminalOutput((prev) => [
        ...prev,
        `Connected to ${podName}.`,
        "root@container:~# ",
      ]);
    }, 2000);

    return () => clearTimeout(timer);
  }, [podName]);

  return (
    <>
      <DialogHeader className="flex h-[60px] flex-row items-center justify-start border-b px-5">
        <DialogTitle className="text-lg font-semibold">
          Web Terminal
        </DialogTitle>
      </DialogHeader>
      <div className="mx-5 grid h-[calc(100vh_-188px)] w-[calc(100vw_-144px)] gap-5 pb-5 md:grid-cols-3 xl:grid-cols-4">
        <Card
          className="h-full overflow-hidden rounded-md bg-gray-900 p-1 text-white dark:border md:col-span-2 xl:col-span-3"
          ref={refRoot}
        >
          <ScrollArea style={{ width, height }} className="p-3">
            {terminalOutput.map((line, index) => (
              <p key={index} className="font-mono text-sm">
                {line}
              </p>
            ))}
          </ScrollArea>
        </Card>
        <div className="space-y-4">
          <ContainerSelect />
          <div className="space-y-2">
            <h4 className="font-semibold">基本信息</h4>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div>状态:</div>
              <div>运行中</div>
              <div>镜像:</div>
              <div>registry.k8s.io/nfd/node-feature-discovery:v0.15.4</div>
              <div>命名:</div>
              <div>nfd-worker</div>
              <div>资源预算:</div>
              <div></div>
              <div>资源上限:</div>
              <div></div>
              <div>重启次数:</div>
              <div>0</div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

export default function TerminalDialog({
  // props
  isOpen,
  setIsOpen,
  podName,
}: {
  // props types
  isOpen: boolean;
  setIsOpen: (value: boolean) => void;
  podName: string;
}) {
  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      {/*  */}
      <DialogContent className="h-[calc(100vh_-104px)] w-[calc(100vw_-104px)] max-w-full gap-5 p-0">
        <Content podName={podName} />
      </DialogContent>
    </Dialog>
  );
}
