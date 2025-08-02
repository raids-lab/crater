import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/monitor')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.clusterMonitoring'),
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
