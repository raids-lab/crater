import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/admin/jobs')({
  component: RouteComponent,
  loader: () => ({ crumb: t('navigation.jobManagement') }),
})

function RouteComponent() {
  return <Outlet />
}
