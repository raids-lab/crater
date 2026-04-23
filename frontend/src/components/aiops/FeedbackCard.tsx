'use client'

import { Check, ChevronDown, ThumbsDown, ThumbsUp } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Slider } from '@/components/ui/slider'
import { Textarea } from '@/components/ui/textarea'

import {
  apiEnrichFeedback,
  apiQuickSubmitFeedback,
  apiSubmitFeedback,
  apiUpsertFeedback,
} from '@/services/api/agent'
import type { AgentFeedback } from '@/services/api/agent'

// ── Predefined tags ─────────────────────────────────────────────────────────

const FEEDBACK_TAGS = [
  { key: 'helpful', label: '很有帮助' },
  { key: 'clear', label: '表述清晰' },
  { key: 'inaccurate', label: '不准确' },
  { key: 'irrelevant', label: '不相关' },
  { key: 'incomplete', label: '不完整' },
  { key: 'wrong_direction', label: '思路方向错误' },
  { key: 'too_slow', label: '响应太慢' },
] as const

const DIMENSION_LABELS: Record<string, string> = {
  relevance: '相关性',
  accuracy: '准确性',
  usefulness: '有用性',
}

const DIMENSION_KEYS = ['relevance', 'accuracy', 'usefulness'] as const

// ── Props ───────────────────────────────────────────────────────────────────

interface FeedbackCardProps {
  sessionId: string
  targetType: 'message' | 'turn'
  targetId: string
  existingFeedback?: AgentFeedback | null
  onFeedbackChange?: (feedback: AgentFeedback) => void
}

export function FeedbackCard({
  sessionId,
  targetType,
  targetId,
  existingFeedback,
  onFeedbackChange,
}: FeedbackCardProps) {
  const { t } = useTranslation()

  // Local state derived from existing feedback or fresh
  const [rating, setRating] = useState<1 | -1 | null>(existingFeedback?.rating ?? null)
  const [selectedTags, setSelectedTags] = useState<string[]>(existingFeedback?.tags ?? [])
  // CHANGED: empty object — no pre-fill with defaults
  const [dimensions, setDimensions] = useState<Record<string, number>>(
    existingFeedback?.dimensions ?? {}
  )
  const [comment, setComment] = useState(existingFeedback?.comment ?? '')
  const [status, setStatus] = useState<'draft' | 'submitted'>(existingFeedback?.status ?? 'draft')
  const [detailsOpen, setDetailsOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  // Track whether the quick-submit path was used
  const [quickSubmitted, setQuickSubmitted] = useState(
    existingFeedback?.status === 'submitted'
  )

  const isSubmitted = status === 'submitted'

  // ── Debounced enrich auto-save ────────────────────────────────────────────

  const enrichTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // When in quick-submitted mode, auto-save enrichment details on any change
  useEffect(() => {
    if (!quickSubmitted) return
    if (!rating) return

    if (enrichTimerRef.current) clearTimeout(enrichTimerRef.current)
    enrichTimerRef.current = setTimeout(async () => {
      setSaving(true)
      try {
        const res = await apiEnrichFeedback({
          sessionId,
          targetType,
          targetId,
          tags: selectedTags,
          dimensions,
          comment,
        })
        if (res?.data) {
          onFeedbackChange?.(res.data)
        }
      } catch {
        // silently handle
      } finally {
        setSaving(false)
      }
    }, 300)

    return () => {
      if (enrichTimerRef.current) clearTimeout(enrichTimerRef.current)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedTags, dimensions, comment, quickSubmitted])

  // ── Handlers ──────────────────────────────────────────────────────────────

  const handleRating = useCallback(
    async (newRating: 1 | -1) => {
      // Once quick-submitted, rating is locked
      if (isSubmitted && quickSubmitted) return

      // Toggle off if clicking same rating
      const effectiveRating = rating === newRating ? null : newRating
      if (effectiveRating === null) {
        setRating(null)
        setDetailsOpen(false)
        return
      }

      setRating(effectiveRating)
      setSaving(true)
      try {
        const res = await apiQuickSubmitFeedback({
          sessionId,
          targetType,
          targetId,
          rating: effectiveRating,
        })
        if (res?.data) {
          setQuickSubmitted(true)
          setStatus('submitted')
          onFeedbackChange?.(res.data)
        }
      } catch {
        // silently handle
      } finally {
        setSaving(false)
      }
    },
    [isSubmitted, quickSubmitted, rating, sessionId, targetType, targetId, onFeedbackChange]
  )

  const toggleTag = useCallback(
    (tag: string) => {
      if (isSubmitted && !quickSubmitted) return
      setSelectedTags((prev) => (prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag]))
    },
    [isSubmitted, quickSubmitted]
  )

  const handleDimensionChange = useCallback(
    (key: string, value: number[]) => {
      if (isSubmitted && !quickSubmitted) return
      setDimensions((prev) => ({ ...prev, [key]: value[0] }))
    },
    [isSubmitted, quickSubmitted]
  )

  const handleCommentChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      if (isSubmitted && !quickSubmitted) return
      setComment(e.target.value)
    },
    [isSubmitted, quickSubmitted]
  )

  // Draft save — only applicable when NOT quickSubmitted
  const handleSaveDraft = useCallback(async () => {
    if (isSubmitted || rating === null) return
    setSaving(true)
    try {
      const res = await apiUpsertFeedback({
        sessionId,
        targetType,
        targetId,
        rating,
        tags: selectedTags,
        dimensions,
        comment,
      })
      if (res?.data) {
        onFeedbackChange?.(res.data)
      }
    } catch {
      // ignore
    } finally {
      setSaving(false)
    }
  }, [isSubmitted, rating, sessionId, targetType, targetId, selectedTags, dimensions, comment, onFeedbackChange])

  // Explicit submit — only applicable when NOT quickSubmitted
  const handleSubmit = useCallback(async () => {
    if (isSubmitted || rating === null) return

    setSaving(true)
    setSubmitting(true)
    try {
      await apiUpsertFeedback({
        sessionId,
        targetType,
        targetId,
        rating,
        tags: selectedTags,
        dimensions,
        comment,
      })
      const res = await apiSubmitFeedback(sessionId, targetType, targetId)
      if (res?.data) {
        setStatus('submitted')
        onFeedbackChange?.(res.data)
      }
    } catch {
      // ignore
    } finally {
      setSaving(false)
      setSubmitting(false)
    }
  }, [isSubmitted, rating, sessionId, targetType, targetId, selectedTags, dimensions, comment, onFeedbackChange])

  // Whether detail fields are editable (editable when: not submitted at all, OR quick-submitted for enrichment)
  const detailsEditable = !isSubmitted || quickSubmitted

  return (
    <div className="mt-1.5 flex flex-col gap-1">
      {/* Thumbs row */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          className={`h-7 w-7 p-0 ${rating === 1 ? 'text-green-600 bg-green-50 dark:bg-green-950/30' : 'text-muted-foreground'}`}
          onClick={() => handleRating(1)}
          disabled={isSubmitted && quickSubmitted}
          title={t('aiops.feedback.thumbsUp', '有帮助')}
        >
          <ThumbsUp className="size-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className={`h-7 w-7 p-0 ${rating === -1 ? 'text-red-600 bg-red-50 dark:bg-red-950/30' : 'text-muted-foreground'}`}
          onClick={() => handleRating(-1)}
          disabled={isSubmitted && quickSubmitted}
          title={t('aiops.feedback.thumbsDown', '没帮助')}
        >
          <ThumbsDown className="size-3.5" />
        </Button>
        {saving && (
          <span className="text-muted-foreground ml-1 text-xs">
            {t('aiops.feedback.saving', '保存中...')}
          </span>
        )}
        {!saving && quickSubmitted && (
          <span className="text-muted-foreground ml-1 flex items-center gap-1 text-xs">
            <Check className="size-3" />
            {t('aiops.feedback.submitted', '已提交')}
          </span>
        )}
      </div>

      {/* Expandable details — available whenever rating is set (before or after quick-submit) */}
      {rating !== null && (
        <Collapsible open={detailsOpen} onOpenChange={setDetailsOpen}>
          {!detailsOpen && (
            <CollapsibleTrigger asChild>
              <Button variant="ghost" size="sm" className="text-muted-foreground h-6 gap-1 px-1 text-xs">
                <ChevronDown className="size-3" />
                {t('aiops.feedback.addDetails', '添加详细反馈')}
              </Button>
            </CollapsibleTrigger>
          )}

          <CollapsibleContent>
            <div className="bg-muted/40 mt-1 space-y-3 rounded-md border p-3">
              {/* Tags */}
              <div>
                <p className="mb-1.5 text-xs font-medium">
                  {t('aiops.feedback.tags', '标签')}
                </p>
                <div className="flex flex-wrap gap-1.5">
                  {FEEDBACK_TAGS.map((tag) => (
                    <Badge
                      key={tag.key}
                      variant={selectedTags.includes(tag.key) ? 'default' : 'outline'}
                      className={`cursor-pointer text-xs ${!detailsEditable ? 'pointer-events-none' : ''}`}
                      onClick={() => toggleTag(tag.key)}
                    >
                      {tag.label}
                    </Badge>
                  ))}
                </div>
              </div>

              {/* Dimensions */}
              <div>
                <p className="mb-1.5 text-xs font-medium">
                  {t('aiops.feedback.dimensions', '维度评分')}
                </p>
                <div className="space-y-2">
                  {DIMENSION_KEYS.map((key) => (
                    <div key={key} className="flex items-center gap-2">
                      <span className="w-14 text-xs">{DIMENSION_LABELS[key]}</span>
                      <Slider
                        min={1}
                        max={5}
                        step={1}
                        value={[dimensions[key] ?? 3]}
                        onValueChange={(v) => handleDimensionChange(key, v)}
                        disabled={!detailsEditable}
                        className="flex-1"
                      />
                      <span className="text-muted-foreground w-4 text-right text-xs">
                        {dimensions[key] !== undefined ? dimensions[key] : '—'}
                      </span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Comment */}
              <div>
                <p className="mb-1.5 text-xs font-medium">
                  {t('aiops.feedback.comment', '补充说明')}
                </p>
                <Textarea
                  placeholder={t('aiops.feedback.commentPlaceholder', '请描述具体问题或建议...')}
                  value={comment}
                  onChange={handleCommentChange}
                  disabled={!detailsEditable}
                  className="min-h-[60px] text-xs"
                  rows={2}
                />
              </div>

              {/* Actions — only shown in draft flow (not quick-submitted) */}
              {!quickSubmitted && !isSubmitted && (
                <div className="flex items-center justify-end gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleSaveDraft}
                    disabled={saving}
                    className="h-7 text-xs"
                  >
                    {t('aiops.feedback.saveDraft', '保存草稿')}
                  </Button>
                  <Button
                    size="sm"
                    onClick={handleSubmit}
                    disabled={submitting}
                    className="h-7 text-xs"
                  >
                    {submitting
                      ? t('aiops.feedback.submitting', '提交中...')
                      : t('aiops.feedback.submit', '提交反馈')}
                  </Button>
                </div>
              )}

              {/* Enrich mode hint — shown after quick-submit when details panel is open */}
              {quickSubmitted && saving && (
                <p className="text-muted-foreground text-right text-xs">
                  {t('aiops.feedback.autoSaving', '自动保存中...')}
                </p>
              )}
            </div>
          </CollapsibleContent>
        </Collapsible>
      )}
    </div>
  )
}
