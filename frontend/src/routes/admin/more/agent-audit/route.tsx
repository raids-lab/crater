import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/admin/more/agent-audit')({
  component: RouteComponent,
  loader: () => ({ crumb: t('navigation.agentAudit') }),
})

function RouteComponent() {
  return <Outlet />
}
