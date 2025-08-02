import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/portal/templates')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.jobTemplates'),
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
