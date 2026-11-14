import { PlusIcon } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import { ServingField, ServingHelp, ServingSelect } from './serving-fields'
import type { ServingFormState } from './use-serving-form'

const selectorOperators = ['In', 'NotIn', 'Exists', 'DoesNotExist']

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
  const newResourceName = resourceName.trim()
  const resourceExists = Object.keys(profile.extendedResources).includes(newResourceName)
  const title = t('kthena.topology.extraConstraints')
  return (
    <section className="grid gap-3">
      <div className="flex flex-wrap items-center gap-1.5">
        <h5 className="text-sm font-medium">{title}</h5>
        <ServingHelp label={title}>{t('kthena.topology.extraConstraintsHelp')}</ServingHelp>
        <Badge variant="secondary" className="font-normal">
          {t('kthena.topology.optional')}
        </Badge>
      </div>
      <div className="grid gap-3">
        {profile.selectors.map((selector, i) => {
          const supported = selectorOperators.includes(selector.operator)
          const hasValues =
            (selector.operator !== 'Exists' && selector.operator !== 'DoesNotExist') ||
            Boolean(selector.values?.length)
          const valuesPath = `${prefix}.selectors.${i}.values` as const
          return (
            <div key={i} className="grid items-start gap-3 rounded-md border p-3 sm:grid-cols-2">
              <ServingField
                form={form}
                name={`${prefix}.selectors.${i}.key`}
                label={t('kthena.topology.selectorLabel')}
                required
              />
              <ServingSelect
                form={form}
                name={`${prefix}.selectors.${i}.operator`}
                label={t('kthena.topology.selectorOperator')}
                options={[
                  ...selectorOperators.map((value) => ({ value, label: value })),
                  ...(!supported && selector.operator
                    ? [{ value: selector.operator, label: selector.operator, disabled: true }]
                    : []),
                ]}
                onChange={(operator) => {
                  if (operator === 'Exists' || operator === 'DoesNotExist')
                    form.setValue(valuesPath, [], { shouldDirty: true })
                }}
              />
              {!supported && (
                <p className="text-destructive text-sm sm:col-span-2">
                  {t('kthena.topology.selectorUnsupported')}
                </p>
              )}
              {hasValues && (
                <FormField
                  control={form.control}
                  name={valuesPath}
                  render={({ field }) => (
                    <FormItem className="sm:col-span-2">
                      <FormLabel>{t('kthena.topology.selectorValues')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          value={(field.value ?? []).join(',')}
                          onChange={(event) => field.onChange(event.target.value.split(','))}
                          onBlur={(event) => {
                            field.onChange(
                              event.target.value
                                .split(',')
                                .map((value) => value.trim())
                                .filter(Boolean)
                            )
                            field.onBlur()
                          }}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
              <Button
                type="button"
                size="sm"
                variant="ghost"
                className="justify-self-end sm:col-span-2"
                onClick={() =>
                  form.setValue(
                    `${prefix}.selectors`,
                    profile.selectors.filter((_, n) => n !== i),
                    { shouldDirty: true }
                  )
                }
              >
                {t('common.delete')}
              </Button>
            </div>
          )
        })}
        {Object.entries(profile.extendedResources).map(([key, value]) => (
          <div
            key={key}
            className="grid items-center gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,8rem)_auto]"
          >
            <span className="font-mono text-sm break-all">{key}</span>
            <Input
              aria-label={key}
              value={value}
              onChange={(event) =>
                form.setValue(
                  `${prefix}.extendedResources`,
                  { ...profile.extendedResources, [key]: event.target.value },
                  { shouldDirty: true }
                )
              }
            />
            <Button
              type="button"
              size="sm"
              variant="ghost"
              className="w-fit"
              onClick={() => {
                const next = { ...profile.extendedResources }
                delete next[key]
                form.setValue(`${prefix}.extendedResources`, next, { shouldDirty: true })
              }}
            >
              {t('common.delete')}
            </Button>
          </div>
        ))}
        <div className="flex flex-wrap items-start gap-2">
          <Button
            type="button"
            variant="secondary"
            onClick={() =>
              form.setValue(
                `${prefix}.selectors`,
                [...profile.selectors, { key: '', operator: 'In', values: [] }],
                { shouldDirty: true }
              )
            }
          >
            <PlusIcon className="size-4" />
            {t('kthena.topology.addSelector')}
          </Button>
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <Input
              aria-label={t('kthena.topology.resourceName')}
              placeholder="vendor.example/resource"
              value={resourceName}
              className="w-full sm:w-52"
              onChange={(event) => setResourceName(event.target.value)}
            />
            <Button
              type="button"
              variant="secondary"
              disabled={!newResourceName || resourceExists}
              onClick={() => {
                form.setValue(
                  `${prefix}.extendedResources`,
                  { ...profile.extendedResources, [newResourceName]: '1' },
                  { shouldDirty: true }
                )
                setResourceName('')
              }}
            >
              <PlusIcon className="size-4" />
              {t('kthena.topology.addResource')}
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
