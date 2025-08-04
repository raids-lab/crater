import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/admin/env/registry')({
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
