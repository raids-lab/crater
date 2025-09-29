/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { useState } from 'react'

import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { PodIngressDialogProps } from '@/components/codeblock/pod-container-dialog'

import useNamespacedState from '@/hooks/use-namespaced-state'

import { IngressPanel } from './ingress-panel'
import { NodeportPanel } from './nodeport-panel'

export default function PodIngressDialog({
  namespacedName,
  setNamespacedName,
  userName,
  jobName,
}: PodIngressDialogProps) {
  const [isOpen, setIsOpen] = useNamespacedState(namespacedName, setNamespacedName)
  const [activeTab, setActiveTab] = useState<'ingress' | 'nodeport'>('ingress')

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
              {namespacedName && (
                <IngressPanel namespacedName={namespacedName} jobName={jobName} />
              )}
            </TabsContent>

            {/* NodePort Tab Content */}
            <TabsContent
              value="nodeport"
              className="grid gap-4 h-full pt-2"
              style={{ display: activeTab === "nodeport" ? "block" : "none" }}
            >
              {namespacedName && (
                <NodeportPanel namespacedName={namespacedName} userName={userName} />
              )}
            </TabsContent>
          </Tabs>
        </div>
      </DialogContent>
    </Dialog>
  )
}
