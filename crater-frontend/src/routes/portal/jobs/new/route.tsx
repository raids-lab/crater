import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/jobs/new')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.jobNew'),
      back: true, // history back when clicking this breadcrumb
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
