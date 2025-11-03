import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/data/models')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.models'),
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
