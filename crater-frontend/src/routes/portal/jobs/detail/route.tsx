import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/jobs/detail')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.jobDetail'),
      back: true, // history back when clicking this breadcrumb
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
