import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { PodIngressDialogProps } from "@/components/codeblock/PodContainerDialog";
import { useNamespacedState } from "@/hooks/useNamespacedState";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { IngressPanel } from "./IngressPanel";
import { NodeportPanel } from "./NodeportPanel";

export default function PodIngressDialog({
  namespacedName,
  setNamespacedName,
  userName,
}: PodIngressDialogProps) {
  const [isOpen, setIsOpen] = useNamespacedState(
    namespacedName,
    setNamespacedName,
  );
  const [activeTab, setActiveTab] = useState<"ingress" | "nodeport">(
    "nodeport",
  );

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      {/* prettier-ignore */}
      <DialogContent className="flex flex-col sm:h-auto overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center justify-start">
            外部访问规则
          </DialogTitle>
        </DialogHeader>

        <div className="grow overflow-y-auto">
          <Tabs
            defaultValue="nodeport"
            value={activeTab}
            onValueChange={(value) =>
              setActiveTab(value as "ingress" | "nodeport")
            }
            className="w-full"
          >
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="ingress">Ingress 规则</TabsTrigger>
              <TabsTrigger value="nodeport">NodePort 规则</TabsTrigger>
            </TabsList>
            
            {/* Ingress Tab Content */}
            <TabsContent
              value="ingress"
              className="grid gap-4 h-full pt-2"
              style={{ display: activeTab === "ingress" ? "block" : "none" }}
            >
              {namespacedName && <IngressPanel namespacedName={namespacedName} />}
            </TabsContent>

            {/* NodePort Tab Content */}
            <TabsContent
              value="nodeport"
              className="grid gap-4 h-full pt-2"
              style={{ display: activeTab === "nodeport" ? "block" : "none" }}
            >
              {namespacedName && <NodeportPanel namespacedName={namespacedName} userName={userName} />}
            </TabsContent>
          </Tabs>
        </div>
      </DialogContent>
    </Dialog>
  );
}
