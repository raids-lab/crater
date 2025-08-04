import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/admin/cluster/resources')({
  loader: () => ({ crumb: t('navigation.resourceManagement') }),
  component: RouteComponent,
})

function RouteComponent() {
  return <Outlet />
}
