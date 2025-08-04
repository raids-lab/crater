import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/admin/cluster/nodes')({
  loader: () => ({ crumb: t('navigation.nodeManagement') }),
  component: RouteComponent,
})

function RouteComponent() {
  return <Outlet />
}
