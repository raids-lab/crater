import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { PlayCircleIcon, RefreshCwIcon } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import { NothingCore } from '@/components/placeholder/nothing'

import {
  AgentAuditTurn,
  AgentQualityEval,
  AgentQualityEvalScope,
  AgentQualityEvalType,
  apiAdminListSessionQualityEvals,
  apiAdminTriggerSessionQualityEval,
} from '@/services/api/admin/agentAudit'

interface Props {
  sessionId: string
  turns?: AgentAuditTurn[]
}

const DIALOGUE_MODEL_ROLES = ['dialogue_eval_flash', 'dialogue_eval_pro'] as const
const TASK_MODEL_ROLES = ['task_eval', 'default', 'coordinator'] as const

function renderScores(scores: unknown) {
  if (!scores || typeof scores !== 'object') return null
  const entries = Object.entries(scores as Record<string, unknown>).filter(
    ([, v]) => typeof v === 'number' || typeof v === 'string'
  )
  if (entries.length === 0) return null
  return (
    <div className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
      {entries.map(([k, v]) => (
        <div
          key={k}
          className="bg-muted/30 border-border/40 flex items-center justify-between rounded-md border px-3 py-2"
        >
          <span className="text-muted-foreground">{k}</span>
          <span className="font-mono">{String(v)}</span>
        </div>
      ))}
    </div>
  )
}

function hasScores(scores: unknown) {
  return !!renderScores(scores)
}

function evalScopeLabel(
  scope: string | undefined,
  t: (key: string, opts?: Record<string, unknown>) => unknown
) {
  return String(
    t(`agentAudit.eval.scope.${scope || 'session'}`, { defaultValue: scope || 'session' })
  )
}

function evalTypeLabel(
  type: string | undefined,
  t: (key: string, opts?: Record<string, unknown>) => unknown
) {
  return String(t(`agentAudit.eval.type.${type || 'full'}`, { defaultValue: type || 'full' }))
}

export function SessionQualityEvalPanel({ sessionId, turns = [] }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [evalScope, setEvalScope] = useState<AgentQualityEvalScope>('session')
  const [evalType, setEvalType] = useState<AgentQualityEvalType>('full')
  const [selectedTurnId, setSelectedTurnId] = useState('')
  const [dialogueModelRole, setDialogueModelRole] = useState<string>('dialogue_eval_flash')
  const [taskModelRole, setTaskModelRole] = useState<string>('task_eval')

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
    mutationFn: () =>
      apiAdminTriggerSessionQualityEval(sessionId, {
        evalScope,
        evalType,
        turnId: evalScope === 'turn' ? selectedTurnId : '',
        dialogueModelRole: evalType === 'task' ? '' : dialogueModelRole,
        taskModelRole: evalType === 'dialogue' ? '' : taskModelRole,
      }),
    onSuccess: () => {
      toast.success(t('agentAudit.eval.triggered'))
      qc.invalidateQueries({ queryKey: ['admin', 'agent-audit', 'quality-evals', sessionId] })
      qc.invalidateQueries({ queryKey: ['admin', 'agent-audit', 'sessions'] })
      qc.invalidateQueries({ queryKey: ['admin', 'agent-audit', 'session-detail', sessionId] })
    },
    onError: (err: unknown) => {
      toast.error(t('agentAudit.eval.triggerFailed', { error: (err as Error).message }))
    },
  })

  useEffect(() => {
    triggerMutation.reset()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId])

  useEffect(() => {
    if (evalScope !== 'turn') return
    if (selectedTurnId && turns.some((turn) => turn.turnId === selectedTurnId)) return
    setSelectedTurnId(turns[0]?.turnId ?? '')
  }, [evalScope, selectedTurnId, turns])

  const evals = evalsQuery.data ?? []
  const latest = evals[0]
  const isPending = evals.some((e) => e.evalStatus === 'pending' || e.evalStatus === 'running')
  const canTrigger = evalScope !== 'turn' || selectedTurnId !== ''
  const selectedTurn = useMemo(
    () => turns.find((turn) => turn.turnId === selectedTurnId),
    [selectedTurnId, turns]
  )

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="space-y-3 p-4">
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <div className="space-y-1">
              <div className="text-muted-foreground text-xs">
                {t('agentAudit.eval.scope.label')}
              </div>
              <Select
                value={evalScope}
                onValueChange={(value) => setEvalScope(value as AgentQualityEvalScope)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="session">{t('agentAudit.eval.scope.session')}</SelectItem>
                  <SelectItem value="turn">{t('agentAudit.eval.scope.turn')}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1">
              <div className="text-muted-foreground text-xs">{t('agentAudit.eval.type.label')}</div>
              <Select
                value={evalType}
                onValueChange={(value) => setEvalType(value as AgentQualityEvalType)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="full">{t('agentAudit.eval.type.full')}</SelectItem>
                  <SelectItem value="dialogue">{t('agentAudit.eval.type.dialogue')}</SelectItem>
                  <SelectItem value="task">{t('agentAudit.eval.type.task')}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {evalScope === 'turn' && (
              <div className="space-y-1">
                <div className="text-muted-foreground text-xs">
                  {t('agentAudit.eval.turn.label')}
                </div>
                <Select value={selectedTurnId} onValueChange={setSelectedTurnId}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder={t('agentAudit.eval.turn.placeholder')} />
                  </SelectTrigger>
                  <SelectContent>
                    {turns.map((turn, index) => (
                      <SelectItem key={turn.turnId} value={turn.turnId}>
                        {t('agentAudit.eval.turn.option', {
                          index: turns.length - index,
                          status: turn.status,
                        })}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            {evalType !== 'task' && (
              <div className="space-y-1">
                <div className="text-muted-foreground text-xs">
                  {t('agentAudit.eval.dialogueModel')}
                </div>
                <Select value={dialogueModelRole} onValueChange={setDialogueModelRole}>
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {DIALOGUE_MODEL_ROLES.map((role) => (
                      <SelectItem key={role} value={role}>
                        {role}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            {evalType !== 'dialogue' && (
              <div className="space-y-1">
                <div className="text-muted-foreground text-xs">
                  {t('agentAudit.eval.taskModel')}
                </div>
                <Select value={taskModelRole} onValueChange={setTaskModelRole}>
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {TASK_MODEL_ROLES.map((role) => (
                      <SelectItem key={role} value={role}>
                        {role}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>

          {evalScope === 'turn' && selectedTurn && (
            <div className="text-muted-foreground text-xs">
              {t('agentAudit.eval.turn.selected', {
                turnId: selectedTurn.turnId.slice(0, 8),
                mode: selectedTurn.orchestrationMode,
              })}
            </div>
          )}

          <div className="flex items-center gap-2">
            <Button
              onClick={() => triggerMutation.mutate()}
              disabled={triggerMutation.isPending || isPending || !canTrigger}
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
              <RefreshCwIcon
                className={`mr-1 h-3.5 w-3.5 ${evalsQuery.isFetching ? 'animate-spin' : ''}`}
              />
              {t('common.refresh')}
            </Button>
          </div>
        </CardContent>
      </Card>

      {latest ? (
        <Card className="border-border shadow-sm">
          <CardHeader className="pb-4">
            <CardTitle className="flex items-center gap-2.5 text-base">
              <Badge
                variant={latest.evalStatus === 'completed' ? 'default' : 'secondary'}
                className="px-2.5 py-0.5 text-xs font-normal"
              >
                {t(`agentAudit.eval.status.${latest.evalStatus}`)}
              </Badge>
              <Badge variant="outline" className="px-2.5 py-0.5 text-xs font-normal">
                {evalScopeLabel(latest.evalScope, t)}
              </Badge>
              <Badge variant="outline" className="px-2.5 py-0.5 text-xs font-normal">
                {evalTypeLabel(latest.evalType, t)}
              </Badge>
              <span className="text-muted-foreground ml-2 text-sm">
                {t(`agentAudit.eval.triggerSource.${latest.triggerSource}`, {
                  defaultValue: latest.triggerSource,
                })}
              </span>
              <span className="text-muted-foreground ml-auto text-sm">
                {new Date(latest.createdAt).toLocaleString()}
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-6 pb-6">
            {hasScores(latest.chatScores) && (
              <div className="space-y-2">
                <div className="flex items-center justify-between gap-2 text-sm">
                  <span>{t('agentAudit.eval.chatScores')}</span>
                  {latest.chatModel && (
                    <span className="text-muted-foreground bg-muted/30 rounded px-2 py-0.5 font-mono text-[13px]">
                      {latest.chatModel}
                    </span>
                  )}
                </div>
                {renderScores(latest.chatScores)}
              </div>
            )}
            {hasScores(latest.chainScores) && (
              <div className="space-y-2">
                <div className="flex items-center justify-between gap-2 text-sm">
                  <span>{t('agentAudit.eval.chainScores')}</span>
                  {latest.chainModel && (
                    <span className="text-muted-foreground bg-muted/30 rounded px-2 py-0.5 font-mono text-[13px]">
                      {latest.chainModel}
                    </span>
                  )}
                </div>
                {renderScores(latest.chainScores)}
              </div>
            )}
            {latest.summary && (
              <div className="space-y-2">
                <div className="text-sm">{t('agentAudit.eval.summary')}</div>
                <pre className="bg-muted/40 border-border/50 text-foreground max-h-[300px] overflow-y-auto rounded-lg border p-4 text-sm leading-relaxed whitespace-pre-wrap">
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
        <Card className="border-border mt-6 shadow-sm">
          <CardHeader className="pb-4">
            <CardTitle className="text-base">{t('agentAudit.eval.history')}</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="max-h-[500px] overflow-y-auto">
              <Accordion type="single" collapsible className="w-full">
                {evals.slice(1).map((e) => (
                  <AccordionItem key={e.id} value={String(e.id)} className="border-border/50 px-4">
                    <AccordionTrigger className="py-4 hover:no-underline">
                      <div className="flex flex-1 items-center gap-3 text-sm">
                        <Badge
                          variant={e.evalStatus === 'completed' ? 'default' : 'secondary'}
                          className="font-normal"
                        >
                          {t(`agentAudit.eval.status.${e.evalStatus}`)}
                        </Badge>
                        <Badge variant="outline" className="font-normal">
                          {evalScopeLabel(e.evalScope, t)}
                        </Badge>
                        <Badge variant="outline" className="font-normal">
                          {evalTypeLabel(e.evalType, t)}
                        </Badge>
                        <span className="text-muted-foreground ml-2 text-sm">
                          {t(`agentAudit.eval.triggerSource.${e.triggerSource}`, {
                            defaultValue: e.triggerSource,
                          })}
                        </span>
                        <span className="text-muted-foreground mr-4 ml-auto text-sm">
                          {new Date(e.createdAt).toLocaleString()}
                        </span>
                      </div>
                    </AccordionTrigger>
                    <AccordionContent className="space-y-6 pt-2 pb-6">
                      {hasScores(e.chatScores) && (
                        <div className="space-y-2">
                          <div className="flex items-center justify-between gap-2 text-sm">
                            <span>{t('agentAudit.eval.chatScores')}</span>
                            {e.chatModel && (
                              <span className="text-muted-foreground bg-muted/30 rounded px-2 py-0.5 font-mono text-[13px]">
                                {e.chatModel}
                              </span>
                            )}
                          </div>
                          {renderScores(e.chatScores)}
                        </div>
                      )}
                      {hasScores(e.chainScores) && (
                        <div className="space-y-2">
                          <div className="flex items-center justify-between gap-2 text-sm">
                            <span>{t('agentAudit.eval.chainScores')}</span>
                            {e.chainModel && (
                              <span className="text-muted-foreground bg-muted/30 rounded px-2 py-0.5 font-mono text-[13px]">
                                {e.chainModel}
                              </span>
                            )}
                          </div>
                          {renderScores(e.chainScores)}
                        </div>
                      )}
                      {e.summary && (
                        <div className="space-y-2">
                          <div className="text-sm">{t('agentAudit.eval.summary')}</div>
                          <pre className="bg-muted/40 border-border/50 text-foreground max-h-[300px] overflow-y-auto rounded-lg border p-4 text-sm leading-relaxed whitespace-pre-wrap">
                            {e.summary}
                          </pre>
                        </div>
                      )}
                    </AccordionContent>
                  </AccordionItem>
                ))}
              </Accordion>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
