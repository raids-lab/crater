import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/data/blocks')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.blocks'),
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
