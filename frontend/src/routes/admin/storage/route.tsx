import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/admin/storage')({
  component: RouteComponent,
  loader: () => ({ crumb: t('navigation.storageManagement') }),
})

function RouteComponent() {
  return <Outlet />
}
