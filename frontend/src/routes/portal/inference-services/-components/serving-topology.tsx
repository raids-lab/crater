import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { ServingField, ServingSelect } from './serving-fields'
import type { ServingFormState } from './use-serving-form'

export function ServingTopologyFields({ state }: { state: ServingFormState }) {
  const { form, values, t, capabilities } = state
  const pd = capabilities.data?.some(
    (c) => c.engine === values.backendType && c.layout === 'disaggregated' && c.enabled
  )
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('kthena.topology.step.topology')}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-5">
        <ServingSelect
          form={form}
          name="backendType"
          disabled={!!state.edit}
          label={t('kthena.topology.engine')}
          options={['vLLM', 'SGLang'].map((value) => ({ value, label: value }))}
          onChange={(v) => state.changeTopology(v as 'vLLM' | 'SGLang', 'combined')}
        />
        <ServingSelect
          form={form}
          name="layout"
          disabled={!!state.edit}
          label={t('kthena.topology.layout')}
          options={[
            { value: 'combined', label: t('kthena.topology.combined') },
            { value: 'disaggregated', label: t('kthena.topology.disaggregated'), disabled: !pd },
          ]}
          onChange={(v) =>
            state.changeTopology(values.backendType, v as 'combined' | 'disaggregated')
          }
        />
        <ServingField
          form={form}
          name="replicas"
          readOnly={!!state.edit}
          numeric
          label={t('kthena.topology.groups')}
        />
        <ServingField
          form={form}
          name="port"
          readOnly={!!state.edit}
          numeric
          label={t('kthena.topology.port')}
        />
        {capabilities.isError ? (
          <div role="alert">
            {t('kthena.topology.capabilityError')}
            <Button type="button" onClick={() => void capabilities.refetch()}>
              {t('kthena.topology.retry')}
            </Button>
          </div>
        ) : (
          <div className="grid gap-2 text-sm">
            {capabilities.data
              ?.filter((c) => c.engine === values.backendType)
              .map((c) => (
                <p key={c.id}>
                  {c.layout} / {c.execution}:{' '}
                  {t(c.enabled ? 'kthena.topology.configured' : 'kthena.topology.disabled')}
                  {!c.enabled && <span className="text-muted-foreground block">{c.reason}</span>}
                </p>
              ))}
          </div>
        )}
        <p className="text-muted-foreground text-sm">{t('kthena.topology.profileHelp')}</p>
      </CardContent>
    </Card>
  )
}
export function ServingPreflightSummary({ state }: { state: ServingFormState }) {
  const { preview, t } = state
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('kthena.topology.step.preview')}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4">
        {!state.previewCurrent && <p role="alert">{t('kthena.topology.stale')}</p>}
        {preview && (
          <>
            <p>
              {preview.queue} · {preview.pods} Pods ·{' '}
              {t(preview.queued ? 'kthena.topology.queued' : 'kthena.topology.admissible')}
            </p>
            <dl className="grid gap-2 text-sm">
              {Object.entries(preview.totalResources).map(([name, value]) => (
                <div key={name} className="flex justify-between gap-2">
                  <dt className="break-all">{name}</dt>
                  <dd>{value}</dd>
                </div>
              ))}
            </dl>
            {preview.warnings.map((w) => (
              <p key={w} className="text-muted-foreground text-sm">
                {w}
              </p>
            ))}
            <details>
              <summary className="cursor-pointer">ModelServing / ModelServer / ModelRoute</summary>
              <pre className="bg-muted mt-3 max-h-96 overflow-auto rounded-lg p-3 text-xs">
                {JSON.stringify(preview.resources, null, 2)}
              </pre>
            </details>
          </>
        )}
      </CardContent>
    </Card>
  )
}
