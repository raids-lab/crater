import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/admin/data')({
  component: RouteComponent,
  loader: () => ({ crumb: t('navigation.dataManagement') }),
})

function RouteComponent() {
  return <Outlet />
}
