import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/admin/accounts')({
  component: RouteComponent,
  loader: () => ({ crumb: t('navigation.accountManagement') }),
})

function RouteComponent() {
  return <Outlet />
}
