import { GlobeIcon, PlusIcon } from 'lucide-react'
import { useFieldArray } from 'react-hook-form'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

import { EnvFormFields } from '@/components/form/env-form-field'
import CardTitle from '@/components/label/card-title'

import { ServingField, ServingHelp } from './serving-fields'
import type { ServingFormState } from './use-serving-form'

function EnvironmentProfile({
  state,
  prefix,
  label,
}: {
  state: ServingFormState
  prefix: `roles.${number}.${'entry' | 'worker'}`
  label?: string
}) {
  const { form, t } = state
  const secrets = useFieldArray({ control: form.control, name: `${prefix}.secretEnvs` })
  const envLabel = t('kthena.form.sections.env')
  const credentialLabel = t('kthena.topology.credentials')

  return (
    <section className="grid min-w-0 gap-4">
      {label && <h4 className="text-muted-foreground text-sm font-medium">{label}</h4>}
      <div className="grid gap-2">
        <div className="flex items-center gap-1.5">
          <h5 className="text-sm font-medium">{envLabel}</h5>
          <ServingHelp label={envLabel}>{t('kthena.topology.environmentHelp')}</ServingHelp>
        </div>
        <EnvFormFields form={form} envPath={`${prefix}.envs`} label={envLabel} />
      </div>
      <div className="grid gap-2">
        <div className="flex items-center gap-1.5">
          <h5 className="text-sm font-medium">{credentialLabel}</h5>
          <ServingHelp label={credentialLabel}>{t('kthena.topology.credentialHelp')}</ServingHelp>
        </div>
        {secrets.fields.map((secret, index) => (
          <div key={secret.id} className="grid gap-3 rounded-md border p-3">
            <ServingField
              form={form}
              name={`${prefix}.secretEnvs.${index}.name`}
              label={t('kthena.topology.envName')}
              placeholder="HF_AUTH_TOKEN"
              required
            />
            <ServingField
              form={form}
              name={`${prefix}.secretEnvs.${index}.secret`}
              label={t('kthena.topology.secretName')}
              placeholder="hf-credentials"
              required
            />
            <ServingField
              form={form}
              name={`${prefix}.secretEnvs.${index}.key`}
              label={t('kthena.topology.secretKey')}
              placeholder="token"
              required
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="justify-self-end"
              onClick={() => secrets.remove(index)}
            >
              {t('kthena.topology.removeEntry')}
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="secondary"
          onClick={() => secrets.append({ name: '', secret: '', key: '' })}
        >
          <PlusIcon className="size-4" />
          {t('kthena.topology.addCredential')}
        </Button>
      </div>
    </section>
  )
}

export function ServingEnvironmentFields({ state }: { state: ServingFormState }) {
  const { t, values } = state
  const profiles = values.roles.flatMap((role, index) => {
    const worker = role.execution === 'ray' && role.worker
    return [
      {
        prefix: `roles.${index}.entry` as const,
        label: worker ? `${role.name} · ${t('kthena.topology.entry')}` : role.name,
      },
      ...(worker
        ? [
            {
              prefix: `roles.${index}.worker` as const,
              label: `${role.name} · ${t('kthena.topology.worker')}`,
            },
          ]
        : []),
    ]
  })

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle icon={GlobeIcon}>{t('kthena.topology.environmentAndCredentials')}</CardTitle>
          <Badge variant="secondary" className="font-normal">
            {t('kthena.topology.optional')}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="grid gap-5 [&>section+section]:border-t [&>section+section]:pt-5">
        {profiles.map(({ prefix, label }) => (
          <EnvironmentProfile
            key={prefix}
            state={state}
            prefix={prefix}
            label={profiles.length > 1 ? label : undefined}
          />
        ))}
      </CardContent>
    </Card>
  )
}
