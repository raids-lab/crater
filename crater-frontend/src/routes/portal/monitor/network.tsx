import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

import NetworkOverview from '@/components/monitors/network-monitor'

export const Route = createFileRoute('/portal/monitor/network')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.networkMonitoring'),
    }
  },
})

function RouteComponent() {
  return <NetworkOverview />
}
