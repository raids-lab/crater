import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { PlayCircleIcon, RefreshCwIcon } from 'lucide-react'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import { NothingCore } from '@/components/placeholder/nothing'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  AgentQualityEval,
  apiAdminListSessionQualityEvals,
  apiAdminTriggerSessionQualityEval,
} from '@/services/api/admin/agentAudit'

interface Props {
  sessionId: string
}

function renderScores(scores: unknown) {
  if (!scores || typeof scores !== 'object') return null
  const entries = Object.entries(scores as Record<string, unknown>).filter(
    ([, v]) => typeof v === 'number' || typeof v === 'string'
  )
  if (entries.length === 0) return null
  return (
    <div className="grid grid-cols-2 gap-1.5 text-xs">
      {entries.map(([k, v]) => (
        <div key={k} className="bg-muted/40 flex items-center justify-between rounded px-2 py-1">
          <span className="text-muted-foreground">{k}</span>
          <span className="font-mono">{String(v)}</span>
        </div>
      ))}
    </div>
  )
}

export function SessionQualityEvalPanel({ sessionId }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const evalsQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'quality-evals', sessionId],
    queryFn: async () => (await apiAdminListSessionQualityEvals(sessionId, 20)).data,
    enabled: !!sessionId,
    refetchInterval: (q) => {
      const data = q.state.data as AgentQualityEval[] | undefined
      if (!data || data.length === 0) return false
      const needsPoll = data.some((e) => e.evalStatus === 'pending' || e.evalStatus === 'running')
      return needsPoll ? 5000 : false
    },
  })

  const triggerMutation = useMutation({
    mutationFn: () => apiAdminTriggerSessionQualityEval(sessionId),
    onSuccess: () => {
      toast.success(t('agentAudit.eval.triggered'))
      qc.invalidateQueries({ queryKey: ['admin', 'agent-audit', 'quality-evals', sessionId] })
      qc.invalidateQueries({ queryKey: ['admin', 'agent-audit', 'sessions'] })
    },
    onError: (err: unknown) => {
      toast.error(t('agentAudit.eval.triggerFailed', { error: (err as Error).message }))
    },
  })

  useEffect(() => {
    triggerMutation.reset()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId])

  const evals = evalsQuery.data ?? []
  const latest = evals[0]
  const isPending = latest?.evalStatus === 'pending' || latest?.evalStatus === 'running'

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Button
          onClick={() => triggerMutation.mutate()}
          disabled={triggerMutation.isPending || isPending}
          size="sm"
        >
          {triggerMutation.isPending || isPending ? (
            <>
              <LoadingCircleIcon className="mr-2 h-4 w-4 animate-spin" />
              {t('agentAudit.eval.triggerWorking')}
            </>
          ) : (
            <>
              <PlayCircleIcon className="mr-2 h-4 w-4" />
              {t('agentAudit.eval.triggerButton')}
            </>
          )}
        </Button>
        <Button variant="ghost" size="sm" onClick={() => evalsQuery.refetch()}>
          <RefreshCwIcon className={`mr-1 h-3.5 w-3.5 ${evalsQuery.isFetching ? 'animate-spin' : ''}`} />
          {t('common.refresh')}
        </Button>
      </div>

      {latest ? (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-sm">
              <Badge variant={latest.evalStatus === 'completed' ? 'default' : 'secondary'}>
                {t(`agentAudit.eval.status.${latest.evalStatus}`)}
              </Badge>
              <span className="text-muted-foreground text-xs">
                {t(`agentAudit.eval.triggerSource.${latest.triggerSource}`, { defaultValue: latest.triggerSource })}
              </span>
              <span className="text-muted-foreground ml-auto text-xs">
                {new Date(latest.createdAt).toLocaleString()}
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {latest.chatScores && (
              <div>
                <div className="mb-1 text-xs font-medium">{t('agentAudit.eval.chatScores')}</div>
                {renderScores(latest.chatScores)}
              </div>
            )}
            {latest.chainScores && (
              <div>
                <div className="mb-1 text-xs font-medium">{t('agentAudit.eval.chainScores')}</div>
                {renderScores(latest.chainScores)}
              </div>
            )}
            {latest.summary && (
              <div>
                <div className="mb-1 text-xs font-medium">{t('agentAudit.eval.summary')}</div>
                <pre className="bg-muted/40 max-h-40 overflow-auto whitespace-pre-wrap rounded p-2 text-xs">
                  {latest.summary}
                </pre>
              </div>
            )}
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="py-8">
            <NothingCore title={t('agentAudit.eval.noRecords')} />
          </CardContent>
        </Card>
      )}

      {evals.length > 1 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs">{t('agentAudit.eval.history')}</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <ScrollArea className="max-h-72">
              <div className="divide-y">
                {evals.slice(1).map((e) => (
                  <div key={e.id} className="flex items-center gap-2 px-3 py-2 text-xs">
                    <Badge variant="outline">{t(`agentAudit.eval.status.${e.evalStatus}`)}</Badge>
                    <span className="text-muted-foreground">
                      {t(`agentAudit.eval.triggerSource.${e.triggerSource}`, { defaultValue: e.triggerSource })}
                    </span>
                    <span className="text-muted-foreground ml-auto">
                      {new Date(e.createdAt).toLocaleString()}
                    </span>
                  </div>
                ))}
              </div>
            </ScrollArea>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
