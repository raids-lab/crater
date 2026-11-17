import { useQuery } from '@tanstack/react-query'
import { PauseIcon, PlayIcon, RefreshCcwIcon } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Card } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { CopyButton } from '@/components/button/copy-button'
import TooltipButton from '@/components/button/tooltip-button'
import { NothingCore } from '@/components/placeholder/nothing'

import {
  KthenaRuntimePod,
  KthenaService,
  apiFollowServingLogs,
  apiServingLogs,
} from '@/services/api/inference'

import { useInferenceKeys } from '@/hooks/inference/use-inference-keys'

const terminalColors = new RegExp(String.fromCharCode(27) + String.raw`\[[0-9;]*m`, 'g')

export function ServingLogDialog({
  service,
  pod,
  onClose,
}: {
  service: KthenaService
  pod: KthenaRuntimePod
  onClose: () => void
}) {
  const { t } = useTranslation()
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>{t('kthena.runtime.logs')}</DialogTitle>
          <DialogDescription className="font-mono break-all">{pod.name}</DialogDescription>
        </DialogHeader>
        <ServingPodLog service={service} pod={pod} />
      </DialogContent>
    </Dialog>
  )
}

export function ServingPodLog({ service, pod }: { service: KthenaService; pod: KthenaRuntimePod }) {
  const { t } = useTranslation()
  const keys = useInferenceKeys()
  const [container, setContainer] = useState(
    pod.stage === 'downloading' ? 'model-downloader' : 'engine'
  )
  const [following, setFollowing] = useState(false)
  const [streamedLogs, setStreamedLogs] = useState<string>()
  const [streamFailed, setStreamFailed] = useState(false)
  const logView = useRef<HTMLPreElement>(null)
  const canReadLogs =
    !!pod.nodeName && (container === 'model-downloader' || pod.phase !== 'Pending')
  const logs = useQuery({
    queryKey: [...keys.detail(service.name), 'logs', pod.name, container],
    queryFn: () => apiServingLogs(service.name, pod.name, container).then((r) => r.data),
    enabled: canReadLogs && !following,
  })
  useEffect(() => {
    if (!following || !canReadLogs) return
    const controller = new AbortController()
    const maxLogLength = 256 << 10
    void apiFollowServingLogs(service.name, pod.name, container, controller.signal, (chunk) => {
      setStreamedLogs((current) => ((current ?? '') + chunk).slice(-maxLogLength))
    })
      .catch(() => {
        if (!controller.signal.aborted) setStreamFailed(true)
      })
      .finally(() => {
        if (!controller.signal.aborted) setFollowing(false)
      })
    return () => controller.abort()
  }, [canReadLogs, container, following, pod.name, service.name])
  useEffect(() => {
    if (following && logView.current) logView.current.scrollTop = logView.current.scrollHeight
  }, [following, streamedLogs])
  const logText = (streamedLogs || logs.data)?.replace(terminalColors, '')
  const followLabel = t(following ? 'kthena.runtime.pauseLogs' : 'kthena.runtime.followLogs')
  return (
    <Card className="bg-muted/40 flex min-h-72 gap-2 p-3 shadow-xs">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-muted-foreground text-sm">{t('kthena.runtime.logsTitle')}</span>
        <div className="flex items-center gap-1">
          {service.modelSource === 'external' && (
            <Select
              value={container}
              onValueChange={(value) => {
                setFollowing(false)
                setStreamedLogs(undefined)
                setStreamFailed(false)
                setContainer(value)
              }}
            >
              <SelectTrigger className="w-44" aria-label={t('kthena.runtime.container')}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="engine">{t('kthena.runtime.engine')}</SelectItem>
                <SelectItem value="model-downloader">{t('kthena.runtime.downloader')}</SelectItem>
              </SelectContent>
            </Select>
          )}
          <TooltipButton
            variant="outline"
            size="icon"
            className="size-8"
            aria-label={t('common.refresh')}
            tooltipContent={t('common.refresh')}
            disabled={!canReadLogs || following || logs.isFetching}
            onClick={() => {
              setStreamedLogs(undefined)
              setStreamFailed(false)
              void logs.refetch()
            }}
          >
            <RefreshCcwIcon className="size-4" />
          </TooltipButton>
          {logText && <CopyButton variant="outline" className="size-8" content={logText} />}
          <TooltipButton
            variant="outline"
            size="icon"
            className="size-8"
            disabled={!canReadLogs}
            aria-label={followLabel}
            tooltipContent={followLabel}
            onClick={() => {
              if (!following) {
                setStreamedLogs(undefined)
                setStreamFailed(false)
              }
              setFollowing(!following)
            }}
          >
            {following ? <PauseIcon className="size-4" /> : <PlayIcon className="size-4" />}
          </TooltipButton>
        </div>
      </div>
      {streamFailed && (
        <p role="alert" className="text-destructive text-sm">
          {t('kthena.runtime.followFailed')}
        </p>
      )}
      {logs.isError && !following && !logText ? (
        <p role="alert" className="text-destructive text-sm break-words">
          {logs.error.message}
        </p>
      ) : logText ? (
        <pre
          ref={logView}
          className="max-h-[55vh] overflow-auto font-mono text-xs break-words whitespace-pre-wrap"
        >
          {logText}
        </pre>
      ) : (
        <NothingCore
          className="min-h-48 flex-1"
          title={t(logs.isLoading ? 'common.loading' : 'kthena.runtime.noLogs')}
        />
      )}
    </Card>
  )
}
