import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/settings')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.userSettings'),
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
