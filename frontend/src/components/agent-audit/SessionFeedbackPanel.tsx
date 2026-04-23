import { useQuery } from '@tanstack/react-query'
import { ThumbsDownIcon, ThumbsUpIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import { NothingCore } from '@/components/placeholder/nothing'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { AgentFeedback, apiAdminListSessionFeedbacks } from '@/services/api/admin/agentAudit'

interface Props {
  sessionId: string
}

function dimensionsList(d: unknown): [string, number][] {
  if (!d || typeof d !== 'object') return []
  return Object.entries(d as Record<string, unknown>)
    .filter(([, v]) => typeof v === 'number')
    .map(([k, v]) => [k, v as number])
}

function tagsList(t: unknown): string[] {
  if (Array.isArray(t)) return t.filter((x): x is string => typeof x === 'string')
  if (t && typeof t === 'object') return Object.keys(t as Record<string, unknown>)
  return []
}

export function SessionFeedbackPanel({ sessionId }: Props) {
  const { t } = useTranslation()
  const feedbacksQuery = useQuery({
    queryKey: ['admin', 'agent-audit', 'feedbacks', sessionId],
    queryFn: async () => (await apiAdminListSessionFeedbacks(sessionId)).data as AgentFeedback[],
    enabled: !!sessionId,
  })

  if (feedbacksQuery.isLoading) {
    return (
      <Card>
        <CardContent className="text-muted-foreground py-12 text-center">
          <LoadingCircleIcon className="mx-auto h-6 w-6 animate-spin" />
        </CardContent>
      </Card>
    )
  }

  const items = feedbacksQuery.data ?? []
  if (items.length === 0) {
    return (
      <Card>
        <CardContent className="py-12">
          <NothingCore title={t('agentAudit.feedback.empty')} />
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-3">
      {items.map((fb) => {
        const dims = dimensionsList(fb.dimensions)
        const tags = tagsList(fb.tags)
        return (
          <Card key={fb.id}>
            <CardContent className="space-y-2 pt-4">
              <div className="flex items-center gap-2 text-sm">
                {fb.rating === 1 ? (
                  <Badge className="gap-1 border-emerald-200 bg-emerald-50 text-emerald-700">
                    <ThumbsUpIcon className="h-3 w-3" /> {t('agentAudit.feedback.ratingUp')}
                  </Badge>
                ) : (
                  <Badge className="gap-1 border-rose-200 bg-rose-50 text-rose-700">
                    <ThumbsDownIcon className="h-3 w-3" /> {t('agentAudit.feedback.ratingDown')}
                  </Badge>
                )}
                <span className="text-muted-foreground text-xs">{fb.targetType}:{fb.targetId}</span>
                <span className="text-muted-foreground ml-auto text-xs">
                  {fb.submittedAt ? t('agentAudit.feedback.submittedAt', { time: new Date(fb.submittedAt).toLocaleString() }) : '-'}
                </span>
              </div>
              {tags.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  <span className="text-muted-foreground text-xs">{t('agentAudit.feedback.tags')}：</span>
                  {tags.map((tag) => (
                    <Badge key={tag} variant="secondary" className="text-[10px]">{tag}</Badge>
                  ))}
                </div>
              )}
              {dims.length > 0 && (
                <div>
                  <div className="text-muted-foreground mb-1 text-xs">{t('agentAudit.feedback.dimensions')}</div>
                  <div className="grid grid-cols-2 gap-1 text-xs">
                    {dims.map(([k, v]) => (
                      <div key={k} className="bg-muted/40 flex justify-between rounded px-2 py-1">
                        <span>{k}</span>
                        <span className="font-mono">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              {fb.comment && (
                <div>
                  <div className="text-muted-foreground mb-1 text-xs">{t('agentAudit.feedback.comment')}</div>
                  <pre className="bg-muted/40 max-h-40 overflow-auto whitespace-pre-wrap rounded p-2 text-xs">{fb.comment}</pre>
                </div>
              )}
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
