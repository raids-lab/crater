import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/overview')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.platformFullName'),
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
