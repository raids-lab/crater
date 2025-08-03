import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/env/registry')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.imageCreation'),
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
