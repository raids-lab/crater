import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/data/datasets')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.datasets'),
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
