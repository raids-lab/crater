import { createFileRoute } from '@tanstack/react-router'

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { HealthOverviewAdmin } from '@/components/aiops/HealthOverviewAdmin'
import { OpsReportTab } from '@/components/aiops/OpsReportTab'

export const Route = createFileRoute('/admin/aiops/')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <Tabs defaultValue="health" className="space-y-4">
      <TabsList>
        <TabsTrigger value="health">健康概览</TabsTrigger>
        <TabsTrigger value="report">智能巡检报告</TabsTrigger>
      </TabsList>
      <TabsContent value="health">
        <HealthOverviewAdmin />
      </TabsContent>
      <TabsContent value="report">
        <OpsReportTab />
      </TabsContent>
    </Tabs>
  )
}
