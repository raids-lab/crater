import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

import ResourseOverview from '@/components/monitors/idle-monitor'

export const Route = createFileRoute('/admin/monitor/idle')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.freeResources'),
    }
  },
})

function RouteComponent() {
  return <ResourseOverview />
}
