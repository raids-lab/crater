import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/users')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.userManagement'),
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
