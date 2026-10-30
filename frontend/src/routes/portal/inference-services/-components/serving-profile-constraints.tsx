import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { ServingField, ServingSelect } from './serving-fields'
import type { ServingFormState } from './use-serving-form'

export function ServingProfileConstraints({
  state,
  index,
  kind,
}: {
  state: ServingFormState
  index: number
  kind: 'entry' | 'worker'
}) {
  const { form, values, t } = state
  const prefix = `roles.${index}.${kind}` as const
  const profile = values.roles[index][kind]
  const [resourceName, setResourceName] = useState('')
  if (!profile) return null
  return (
    <details>
      <summary>{t('kthena.topology.extraConstraints')}</summary>
      <div className="mt-3 grid gap-3">
        {profile.selectors.map((selector, i) => (
          <div key={i} className="grid gap-2 rounded border p-3">
            <ServingField form={form} name={`${prefix}.selectors.${i}.key`} label="Label" />
            <ServingSelect
              form={form}
              name={`${prefix}.selectors.${i}.operator`}
              label="Operator"
              options={['In', 'NotIn', 'Exists', 'DoesNotExist', 'Gt', 'Lt'].map((value) => ({
                value,
                label: value,
              }))}
            />
            <Input
              aria-label={t('kthena.topology.selectorValues')}
              value={(selector.values ?? []).join(',')}
              onChange={(e) =>
                form.setValue(
                  `${prefix}.selectors.${i}.values`,
                  e.target.value ? e.target.value.split(',') : []
                )
              }
            />
            <Button
              type="button"
              variant="ghost"
              onClick={() =>
                form.setValue(
                  `${prefix}.selectors`,
                  profile.selectors.filter((_, n) => n !== i)
                )
              }
            >
              {t('kthena.form.template.delete')}
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            form.setValue(`${prefix}.selectors`, [
              ...profile.selectors,
              { key: '', operator: 'In', values: [] },
            ])
          }
        >
          {t('kthena.topology.addSelector')}
        </Button>
        {Object.entries(profile.extendedResources).map(([key, value]) => (
          <label key={key} className="grid gap-2 text-sm">
            {key}
            <Input
              aria-label={key}
              value={value}
              onChange={(e) =>
                form.setValue(`${prefix}.extendedResources`, {
                  ...profile.extendedResources,
                  [key]: e.target.value,
                })
              }
            />
            <Button
              type="button"
              variant="ghost"
              onClick={() => {
                const next = { ...profile.extendedResources }
                delete next[key]
                form.setValue(`${prefix}.extendedResources`, next)
              }}
            >
              {t('kthena.form.template.delete')}
            </Button>
          </label>
        ))}
        <Input
          aria-label={t('kthena.topology.resourceName')}
          placeholder="vendor.example/resource"
          value={resourceName}
          onChange={(e) => setResourceName(e.target.value)}
        />
        <Button
          type="button"
          variant="outline"
          disabled={!resourceName.trim()}
          onClick={() => {
            form.setValue(`${prefix}.extendedResources`, {
              ...profile.extendedResources,
              [resourceName.trim()]: '1',
            })
            setResourceName('')
          }}
        >
          {t('kthena.topology.addResource')}
        </Button>
      </div>
    </details>
  )
}
