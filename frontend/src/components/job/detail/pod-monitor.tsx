import { useAtomValue } from 'jotai'
import { useTranslation } from 'react-i18next'

import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'

import GrafanaIframe from '@/components/layout/embed/grafana-iframe'

import { configGrafanaJobAtom } from '@/utils/store/config'

export type PodMonitorTarget = { name: string; nodeName: string }

export function PodMonitorSheet({
  pod,
  onClose,
}: {
  pod: PodMonitorTarget | undefined
  onClose: () => void
}) {
  const { t } = useTranslation()
  const grafana = useAtomValue(configGrafanaJobAtom)
  const base = grafana.pod
  const params = pod
    ? new URLSearchParams({
        orgId: '1',
        refresh: '5s',
        'var-node_name': pod.nodeName,
        'var-pod_name': pod.name,
        'var-gpu': 'All',
        from: 'now-15m',
        to: 'now',
      })
    : undefined

  return (
    <Sheet open={!!pod} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="sm:max-w-4xl">
        <SheetHeader>
          <SheetTitle>{t('common.podMonitor')}</SheetTitle>
        </SheetHeader>
        {pod && params && (
          <div className="h-[calc(100vh-6rem)] w-full px-4">
            <GrafanaIframe baseSrc={`${base}${base.includes('?') ? '&' : '?'}${params}`} />
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}
