import { CircleAlertIcon, ClipboardCheckIcon, WorkflowIcon } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

import CardTitle from '@/components/label/card-title'

import { ServingField, ServingHelp, ServingSelect } from './serving-fields'
import { ServingRoleSummary } from './serving-role-summary'
import type { ServingFormState } from './use-serving-form'

export function ServingTopologyFields({ state }: { state: ServingFormState }) {
  const { form, values, t, capabilities, modelQuery, edit } = state
  const selectedModel = modelQuery.data?.find(({ id }) => id === values.platformModelId)
  const engineProfiles = capabilities.data?.filter(
    (profile) => profile.engine === values.backendType
  )
  const clusterUnavailable =
    capabilities.isError || capabilities.data?.some((profile) => profile.evidence === 'unknown')
  const pd = engineProfiles?.some(
    (profile) => profile.layout === 'disaggregated' && profile.enabled
  )
  const layoutEnabled = engineProfiles?.some(
    (profile) => profile.layout === values.layout && profile.enabled
  )
  const layoutLabel = t(
    values.layout === 'combined' ? 'kthena.topology.combined' : 'kthena.topology.disaggregated'
  )
  return (
    <div className="grid gap-4 md:gap-6">
      <Card>
        <CardHeader>
          <CardTitle icon={WorkflowIcon}>{t('kthena.topology.step.topology')}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-5">
          <div className="grid items-start gap-4 sm:grid-cols-2">
            <ServingSelect
              form={form}
              name="backendType"
              disabled={!!state.edit}
              label={t('kthena.topology.engine')}
              options={['vLLM', 'SGLang'].map((value) => ({ value, label: value }))}
              onValueChange={(value) => {
                const engine = value as 'vLLM' | 'SGLang'
                const supportsLayout = capabilities.data?.some(
                  (profile) =>
                    profile.engine === engine && profile.layout === values.layout && profile.enabled
                )
                state.changeTopology(engine, supportsLayout ? values.layout : 'combined')
              }}
            />
            <ServingSelect
              form={form}
              name="layout"
              disabled={!!state.edit}
              label={t('kthena.topology.layout')}
              help={t(
                values.layout === 'combined'
                  ? 'kthena.topology.combinedHelp'
                  : 'kthena.topology.disaggregatedHelp'
              )}
              options={[
                { value: 'combined', label: t('kthena.topology.combined') },
                {
                  value: 'disaggregated',
                  label:
                    engineProfiles &&
                    !clusterUnavailable &&
                    !pd &&
                    values.layout !== 'disaggregated'
                      ? t('kthena.topology.unavailableOption', {
                          option: t('kthena.topology.disaggregated'),
                        })
                      : t('kthena.topology.disaggregated'),
                  disabled: !pd || clusterUnavailable,
                },
              ]}
              onValueChange={(v) =>
                state.changeTopology(values.backendType, v as 'combined' | 'disaggregated')
              }
            />
          </div>
          <ServingField
            form={form}
            name="replicas"
            readOnly={!!state.edit}
            numeric
            label={t('kthena.topology.groups')}
            help={t('kthena.topology.replicasHelp')}
          />
          {clusterUnavailable ? (
            <div
              className="bg-muted/50 flex flex-wrap items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm"
              role="alert"
            >
              <span className="flex items-center gap-2">
                <CircleAlertIcon className="text-muted-foreground size-4 shrink-0" aria-hidden />
                {t('kthena.topology.capabilityError')}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={capabilities.isFetching}
                onClick={() => void capabilities.refetch()}
              >
                {t('kthena.topology.retry')}
              </Button>
            </div>
          ) : engineProfiles && !layoutEnabled ? (
            <p className="text-destructive text-sm" role="alert">
              {t('kthena.topology.unavailableOption', { option: layoutLabel })}
            </p>
          ) : null}
          <section className="grid gap-3 border-t pt-4">
            <div>
              <div className="flex items-center gap-1.5">
                <h3 className="text-sm font-medium">{t('kthena.topology.modelNameAndPort')}</h3>
                <ServingHelp label={t('kthena.topology.modelNameAndPort')}>
                  {t('kthena.topology.routeModel')}
                </ServingHelp>
              </div>
              {edit && (
                <p className="text-muted-foreground mt-1 text-sm">
                  {t('kthena.topology.runtimeSettingsLocked')}
                </p>
              )}
            </div>
            <div className="grid items-start gap-4 sm:grid-cols-2">
              <ServingField
                form={form}
                name="servedModel"
                readOnly={!!edit}
                label={t('kthena.form.fields.servedModel')}
                placeholder={values.modelSource === 'platform' ? selectedModel?.name : undefined}
                help={t('kthena.topology.engineModelHelp')}
              />
              <ServingField
                form={form}
                name="port"
                readOnly={!!edit}
                numeric
                label={t('kthena.topology.port')}
                help={t('kthena.topology.portHelp')}
              />
            </div>
          </section>
        </CardContent>
      </Card>
    </div>
  )
}
export function ServingPreflightSummary({ state }: { state: ServingFormState }) {
  const { preview, previewCurrent, t, edit, modelQuery } = state
  const model = modelQuery.data?.find((item) => item.id === preview?.spec.platformModelId)
  const help = t(edit ? 'kthena.form.reviewEditHelp' : 'kthena.form.reviewHelp')
  const status = edit
    ? t('kthena.form.reviewEditStatus')
    : t(preview?.queued ? 'kthena.topology.queued' : 'kthena.topology.admissible')

  return (
    <Card>
      <CardHeader>
        <CardTitle icon={ClipboardCheckIcon}>{t('kthena.topology.step.preview')}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4">
        {previewCurrent && preview ? (
          <>
            <div className="bg-muted/40 space-y-2 rounded-md border p-4">
              <Badge variant={preview.queued ? 'secondary' : 'default'}>{status}</Badge>
              <p className="text-muted-foreground text-sm">{help}</p>
            </div>
            <dl className="grid gap-4 text-sm sm:grid-cols-2">
              <div>
                <dt className="text-muted-foreground">{t('kthena.form.fields.serviceName')}</dt>
                <dd className="mt-1 font-medium break-all">{preview.spec.name}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">
                  {t(
                    preview.spec.modelSource === 'platform'
                      ? 'kthena.form.fields.platformModel'
                      : 'kthena.form.fields.modelURI'
                  )}
                </dt>
                <dd className="mt-1 font-medium break-all">
                  {model?.name || preview.spec.modelURI || preview.spec.servedModel}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">{t('kthena.detail.servedModel')}</dt>
                <dd className="mt-1 font-medium break-all">{preview.spec.servedModel || '—'}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">{t('kthena.table.queue')}</dt>
                <dd className="mt-1 font-mono break-all">{preview.queue}</dd>
              </div>
            </dl>
            <section className="grid gap-2">
              <h3 className="text-sm font-medium">{t('kthena.topology.step.roles')}</h3>
              <ServingRoleSummary roles={preview.spec.roles} />
            </section>
            <Dialog>
              <DialogTrigger asChild>
                <Button type="button" variant="outline" size="sm" className="w-fit">
                  {t('kthena.form.viewSpec')}
                </Button>
              </DialogTrigger>
              <DialogContent className="min-w-0 sm:max-w-3xl">
                <DialogHeader>
                  <DialogTitle>{t('kthena.form.viewSpec')}</DialogTitle>
                  <DialogDescription>{t('kthena.detail.orchestrationHelp')}</DialogDescription>
                </DialogHeader>
                <pre className="bg-muted max-h-[60dvh] min-w-0 overflow-auto rounded-md p-3 text-xs">
                  {JSON.stringify(preview.resources, null, 2)}
                </pre>
              </DialogContent>
            </Dialog>
          </>
        ) : (
          <p role="status" className="text-muted-foreground text-sm">
            {t(state.preflight.isPending ? 'kthena.topology.loading' : 'kthena.topology.stale')}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
