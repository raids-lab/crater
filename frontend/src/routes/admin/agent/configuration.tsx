import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { CircleHelpIcon, Loader2Icon, SaveIcon, ShieldCheckIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'

import SimpleTooltip from '@/components/label/simple-tooltip'
import PageTitle from '@/components/layout/page-title'

import {
  type AgentLocaleRuntimeConfig,
  type AgentRuntimeConfig,
  apiAdminGetAgentRuntimeConfig,
  apiAdminUpdateAgentRuntimeConfig,
} from '@/services/api/agent'

export const Route = createFileRoute('/admin/agent/configuration')({
  component: AgentConfigurationPage,
})

type LocaleKey = keyof AgentRuntimeConfig['locales']
type FieldKey = keyof AgentLocaleRuntimeConfig

const fields: Array<{
  key: FieldKey
  labelKey: string
  descriptionKey: string
  rows: number
  maxLength: number
  monospace?: boolean
}> = [
  {
    key: 'basePrompt',
    labelKey: 'agentConfiguration.fields.basePrompt',
    descriptionKey: 'agentConfiguration.fields.basePromptDescription',
    rows: 14,
    maxLength: 100_000,
  },
  {
    key: 'platformGuidePrompt',
    labelKey: 'agentConfiguration.fields.platformGuidePrompt',
    descriptionKey: 'agentConfiguration.fields.platformGuidePromptDescription',
    rows: 16,
    maxLength: 100_000,
  },
  {
    key: 'platformKnowledge',
    labelKey: 'agentConfiguration.fields.platformKnowledge',
    descriptionKey: 'agentConfiguration.fields.platformKnowledgeDescription',
    rows: 24,
    maxLength: 100_000,
  },
  {
    key: 'formAssistantPrompt',
    labelKey: 'agentConfiguration.fields.formAssistantPrompt',
    descriptionKey: 'agentConfiguration.fields.formAssistantPromptDescription',
    rows: 16,
    maxLength: 100_000,
  },
  {
    key: 'jobDiagnosisPrompt',
    labelKey: 'agentConfiguration.fields.jobDiagnosisPrompt',
    descriptionKey: 'agentConfiguration.fields.jobDiagnosisPromptDescription',
    rows: 20,
    maxLength: 100_000,
  },
  {
    key: 'jobDiagnosisUnavailablePrompt',
    labelKey: 'agentConfiguration.fields.jobDiagnosisUnavailablePrompt',
    descriptionKey: 'agentConfiguration.fields.jobDiagnosisUnavailablePromptDescription',
    rows: 6,
    maxLength: 100_000,
  },
  {
    key: 'titlePrompt',
    labelKey: 'agentConfiguration.fields.titlePrompt',
    descriptionKey: 'agentConfiguration.fields.titlePromptDescription',
    rows: 10,
    maxLength: 100_000,
  },
  {
    key: 'scopeRefusalMessage',
    labelKey: 'agentConfiguration.fields.scopeRefusalMessage',
    descriptionKey: 'agentConfiguration.fields.scopeRefusalMessageDescription',
    rows: 5,
    maxLength: 100_000,
  },
  {
    key: 'promptOverrideRegex',
    labelKey: 'agentConfiguration.fields.promptOverrideRegex',
    descriptionKey: 'agentConfiguration.fields.promptOverrideRegexDescription',
    rows: 5,
    maxLength: 8_192,
    monospace: true,
  },
]

function LocaleEditor({
  locale,
  config,
  onChange,
}: {
  locale: LocaleKey
  config: AgentLocaleRuntimeConfig
  onChange: (field: FieldKey, value: string) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="space-y-6 pt-4">
      {fields.map((field) => {
        const inputID = `${locale}-${field.key}`
        return (
          <div key={field.key} className="space-y-2">
            <Label htmlFor={inputID} className="inline-flex items-center gap-1.5 text-base">
              <span>{t(field.labelKey)}</span>
              <SimpleTooltip tooltip={t(field.descriptionKey)} ariaLabel={t(field.descriptionKey)}>
                <CircleHelpIcon className="text-muted-foreground size-4 cursor-help" />
              </SimpleTooltip>
            </Label>
            <p className="text-muted-foreground text-sm">{t(field.descriptionKey)}</p>
            <Textarea
              id={inputID}
              rows={field.rows}
              maxLength={field.maxLength}
              value={config[field.key]}
              spellCheck={!field.monospace}
              className={field.monospace ? 'font-mono text-xs' : 'leading-6'}
              onChange={(event) => onChange(field.key, event.target.value)}
            />
          </div>
        )
      })}
    </div>
  )
}

function AgentConfigurationPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<AgentRuntimeConfig>()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const { data, isError, isLoading, refetch } = useQuery({
    queryKey: ['admin', 'agent', 'runtime-config'],
    queryFn: () => apiAdminGetAgentRuntimeConfig().then((response) => response.data),
  })

  useEffect(() => {
    if (data) setDraft(data)
  }, [data])

  const saveMutation = useMutation({
    mutationFn: apiAdminUpdateAgentRuntimeConfig,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['admin', 'agent', 'runtime-config'] })
      toast.success(t('agentConfiguration.saveSuccess'))
    },
  })

  const updateField = (locale: LocaleKey, field: FieldKey, value: string) => {
    setDraft((current) =>
      current
        ? {
            ...current,
            locales: {
              ...current.locales,
              [locale]: { ...current.locales[locale], [field]: value },
            },
          }
        : current
    )
  }

  const requestSave = () => {
    if (!draft) return
    const emptyField = (Object.keys(draft.locales) as LocaleKey[]).some((locale) =>
      fields.some((field) => !draft.locales[locale][field.key].trim())
    )
    if (emptyField) {
      toast.error(t('agentConfiguration.validationRequired'))
      return
    }
    try {
      new RegExp(draft.locales.zhCN.promptOverrideRegex, 'iu')
      new RegExp(draft.locales.enUS.promptOverrideRegex, 'iu')
    } catch {
      toast.error(t('agentConfiguration.validationRegex'))
      return
    }
    setConfirmOpen(true)
  }

  const save = () => {
    if (!draft) return
    saveMutation.mutate(draft, {
      onSuccess: () => setConfirmOpen(false),
    })
  }

  return (
    <div className="space-y-4 pb-10">
      <PageTitle
        title={t('agentConfiguration.title')}
        description={t('agentConfiguration.description')}
      />
      <Alert>
        <ShieldCheckIcon />
        <AlertTitle>{t('agentConfiguration.safetyTitle')}</AlertTitle>
        <AlertDescription>{t('agentConfiguration.safetyDescription')}</AlertDescription>
      </Alert>

      <Card>
        <CardHeader>
          <CardTitle>{t('agentConfiguration.editorTitle')}</CardTitle>
          <CardDescription>{t('agentConfiguration.editorDescription')}</CardDescription>
        </CardHeader>
        <CardContent>
          {isError ? (
            <div className="flex min-h-48 flex-col items-center justify-center gap-3 text-center">
              <p className="text-destructive text-sm">{t('agentConfiguration.loadError')}</p>
              <Button variant="outline" onClick={() => void refetch()}>
                {t('agentConfiguration.retry')}
              </Button>
            </div>
          ) : isLoading || !draft ? (
            <div className="text-muted-foreground flex min-h-48 items-center justify-center gap-2">
              <Loader2Icon className="size-4 animate-spin" />
              {t('agentConfiguration.loading')}
            </div>
          ) : (
            <Tabs defaultValue="zhCN">
              <TabsList>
                <TabsTrigger value="zhCN">{t('agentConfiguration.localeZhCN')}</TabsTrigger>
                <TabsTrigger value="enUS">{t('agentConfiguration.localeEnUS')}</TabsTrigger>
              </TabsList>
              <TabsContent value="zhCN">
                <LocaleEditor
                  locale="zhCN"
                  config={draft.locales.zhCN}
                  onChange={(field, value) => updateField('zhCN', field, value)}
                />
              </TabsContent>
              <TabsContent value="enUS">
                <LocaleEditor
                  locale="enUS"
                  config={draft.locales.enUS}
                  onChange={(field, value) => updateField('enUS', field, value)}
                />
              </TabsContent>
            </Tabs>
          )}
        </CardContent>
      </Card>

      <div className="bg-background/95 sticky bottom-0 flex justify-end border-t py-4 backdrop-blur">
        <Button onClick={requestSave} disabled={!draft || saveMutation.isPending}>
          {saveMutation.isPending ? <Loader2Icon className="animate-spin" /> : <SaveIcon />}
          {t('agentConfiguration.save')}
        </Button>
      </div>

      <AlertDialog
        open={confirmOpen}
        onOpenChange={(nextOpen) => {
          if (!saveMutation.isPending) setConfirmOpen(nextOpen)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('agentConfiguration.confirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('agentConfiguration.confirmDescription')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={saveMutation.isPending}>
              {t('common.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={saveMutation.isPending}
              onClick={(event) => {
                event.preventDefault()
                save()
              }}
            >
              {saveMutation.isPending ? <Loader2Icon className="animate-spin" /> : <SaveIcon />}
              {t('agentConfiguration.confirmAction')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
