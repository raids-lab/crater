import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

import NvidiaOverview from '@/components/monitors/nvidia-monitor'

export const Route = createFileRoute('/portal/monitor/gpu')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.gpuMonitoring'),
    }
  },
})

function RouteComponent() {
  return <NvidiaOverview />
}
