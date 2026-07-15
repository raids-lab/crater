import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/aiops/')({
  beforeLoad: () => {
    throw redirect({ to: '/admin' })
  },
  component: RouteComponent,
})

function RouteComponent() {
  return null
}

/*
 * 管理端 AI 运维独立页面暂时下线：保留主体代码，后续如需恢复可还原。
 *
 * import { t } from 'i18next'
 * import { AgentAuditSessionsDialog } from '@/components/agent-audit/AgentAuditSessionsDialog'
 * import { HealthOverviewAdmin } from '@/components/aiops/HealthOverviewAdmin'
 * import { OpsReportTab } from '@/components/aiops/OpsReportTab'
 *
 * export const Route = createFileRoute('/admin/aiops/')({
 *   component: RouteComponent,
 *   loader: () => ({ crumb: t('navigation.aiops') }),
 * })
 *
 * function RouteComponent() {
 *   return (
 *     <div className="space-y-4">
 *       <HealthOverviewAdmin actions={<AgentAuditSessionsDialog />} />
 *
 *       <div className="flex items-center justify-between">
 *         <h2 className="text-lg font-semibold">智能巡检报告</h2>
 *       </div>
 *       <OpsReportTab />
 *     </div>
 *   )
 * }
 */
