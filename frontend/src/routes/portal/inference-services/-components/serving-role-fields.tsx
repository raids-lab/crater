import { useQuery } from '@tanstack/react-query'
import { CpuIcon, PlusIcon } from 'lucide-react'
import { useEffect } from 'react'
import { useFieldArray } from 'react-hook-form'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

import { BillingPricePreview } from '@/components/custom/billing-price-preview'
import { NodeMultiSelect } from '@/components/form/other-options-form-field'
import { ResourceFormFields } from '@/components/form/resource-form-field'
import CardTitle from '@/components/label/card-title'

import { apiResourceList, apiResourceNetworks } from '@/services/api/resource'

import { ServingField, ServingHelp, ServingSelect } from './serving-fields'
import { selectedServingImage, servingParallelism } from './serving-form-schema'
import { ServingProfileConstraints } from './serving-profile-constraints'
import { resourceSummary, restoreNetworkProfile } from './serving-serialization'
import type { ServingFormState } from './use-serving-form'

function PodFields({
  state,
  index,
  kind,
}: {
  state: ServingFormState
  index: number
  kind: 'entry' | 'worker'
}) {
  const { form, t, values } = state
  const prefix = `roles.${index}.${kind}` as const
  const profile = values.roles[index][kind]
  const registry = useQuery({
    queryKey: ['resources', 'list'],
    queryFn: () => apiResourceList(true),
  })
  const gpuID =
    registry.data?.data.find((r) => r.name === profile?.resource.gpu.model && r.amountSingleMax > 0)
      ?.ID ?? 0
  const networks = useQuery({
    queryKey: ['resources', 'networks', 'list', gpuID],
    queryFn: () => apiResourceNetworks(gpuID),
    enabled: gpuID > 0,
    select: (r) => r.data.filter((n) => n.amountSingleMax > 0),
  })
  useEffect(() => {
    if (!profile || !networks.data) return
    const restored = restoreNetworkProfile(
      profile,
      networks.data.map((n) => n.name)
    )
    if (restored !== profile) form.setValue(prefix, restored)
  }, [networks.data, profile, form, prefix])
  if (!profile) return null
  const capability = state.capabilities.data?.find(
    (c) =>
      c.engine === values.backendType &&
      c.layout === values.layout &&
      c.execution === values.roles[index].execution
  )
  const selectedImage = selectedServingImage(profile) ?? ''
  const images = capability?.enabled ? capability.images : []
  const imageUnapproved = capability?.enabled && !!selectedImage && !images.includes(selectedImage)
  const imageOptions: { value: string; label: string; disabled?: boolean }[] = images.map(
    (value) => ({ value, label: value })
  )
  if (selectedImage && !images.includes(selectedImage)) {
    imageOptions.push({ value: selectedImage, label: selectedImage, disabled: true })
  }
  const podLabel = t(
    kind === 'worker'
      ? 'kthena.topology.worker'
      : values.roles[index].execution === 'ray'
        ? 'kthena.topology.entry'
        : 'kthena.topology.podResources'
  )

  return (
    <div className="grid gap-4 rounded-lg border p-4">
      <div className="flex items-center gap-1.5">
        <h4 className="text-sm font-medium">{podLabel}</h4>
        <ServingHelp label={podLabel}>{t('kthena.topology.podResourceHelp')}</ServingHelp>
      </div>
      <ResourceFormFields
        form={form}
        allowFractionalCPUAndMemory
        showBillingPricePreview={false}
        cpuPath={`${prefix}.resource.cpu`}
        memoryPath={`${prefix}.resource.memory`}
        gpuCountPath={`${prefix}.resource.gpu.count`}
        gpuModelPath={`${prefix}.resource.gpu.model`}
        rdmaPath={{
          rdmaEnabled: `${prefix}.resource.network.enabled`,
          rdmaCount: `${prefix}.networkCount`,
          rdmaLabel: `${prefix}.resource.network.model`,
        }}
      />
      {profile.resource.network.enabled && (
        <ServingField
          form={form}
          name={`${prefix}.networkCount`}
          numeric
          label={t('kthena.topology.networkCount')}
        />
      )}
      <div className="grid gap-2">
        <ServingSelect
          form={form}
          name={`${prefix}.image`}
          label={t('kthena.topology.imageLabel')}
          help={t('kthena.topology.imageHelp')}
          value={selectedImage}
          placeholder={t('kthena.topology.selectImage')}
          options={imageOptions}
          disabled={images.length === 0}
          required
          onValueChange={(image) =>
            form.setValue(
              prefix,
              { ...profile, imageSource: 'manual', image, imageArchs: [] },
              {
                shouldDirty: true,
                shouldValidate: true,
              }
            )
          }
        />
        {images.length === 0 && !state.capabilities.isPending && (
          <p role="alert" className="text-destructive text-sm">
            {t('kthena.topology.noImages')}
          </p>
        )}
        {capability?.connector && (
          <p className="text-muted-foreground text-sm">KV: {capability.connector}</p>
        )}
        {imageUnapproved && (
          <p role="alert" className="text-destructive text-sm">
            {t('kthena.topology.unapprovedImage')}
          </p>
        )}
      </div>
      <section className="grid gap-3 border-t pt-4">
        <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
          <div className="flex items-center gap-1.5">
            <h4 className="text-sm font-medium">{t('kthena.topology.nodes')}</h4>
            <ServingHelp label={t('kthena.topology.nodes')}>
              {t('kthena.topology.nodeHelp')}
            </ServingHelp>
          </div>
          <label className="flex items-center gap-2 text-sm">
            <Checkbox
              checked={profile.nodeSelector.enable}
              onCheckedChange={(checked) =>
                form.setValue(`${prefix}.nodeSelector.enable`, checked === true, {
                  shouldDirty: true,
                })
              }
            />
            {t('kthena.topology.constrainNodes')}
          </label>
        </div>
        {profile.nodeSelector.enable && (
          <div className="bg-muted/30 grid items-start gap-3 rounded-md p-3 sm:grid-cols-[12rem_minmax(0,1fr)]">
            <ServingSelect
              form={form}
              name={`${prefix}.nodeSelector.mode`}
              label={t('kthena.topology.nodeMode')}
              options={['include', 'exclude'].map((value) => ({
                value,
                label: t(`kthena.topology.${value}`),
              }))}
            />
            <FormField
              control={form.control}
              name={`${prefix}.nodeSelector.nodes`}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('kthena.topology.nodes')}</FormLabel>
                  <NodeMultiSelect
                    values={field.value ?? []}
                    onChange={field.onChange}
                    options={state.nodeQuery.data?.map((node) => node.value) ?? []}
                    placeholder={t(
                      profile.nodeSelector.mode === 'exclude'
                        ? 'otherOptionsFormCard.excludeNodePlaceholder'
                        : 'otherOptionsFormCard.includeNodePlaceholder'
                    )}
                    selectedLabel={t('otherOptionsFormCard.selectedNodeCount', {
                      count: field.value?.length ?? 0,
                    })}
                    emptyLabel={t('otherOptionsFormCard.emptyNodeList')}
                    removeLabel={(name) => t('otherOptionsFormCard.removeNodeAriaLabel', { name })}
                  />
                  <FormMessage />
                </FormItem>
              )}
            />
            {state.nodeQuery.isError && (
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="w-fit"
                onClick={() => void state.nodeQuery.refetch()}
              >
                {t('kthena.topology.retry')}
              </Button>
            )}
          </div>
        )}
        <ServingProfileConstraints state={state} index={index} kind={kind} />
      </section>
    </div>
  )
}
function RoleFields({ state, index }: { state: ServingFormState; index: number }) {
  const { form, t, values } = state
  const role = values.roles[index]
  const { tensorParallel, pipelineParallel } = servingParallelism(role)
  const config = useFieldArray({ control: form.control, name: `roles.${index}.configItems` })
  const rayProfile = state.capabilities.data?.find(
    (c) => c.engine === values.backendType && c.layout === values.layout && c.execution === 'ray'
  )
  const showExecution = rayProfile?.enabled || role.execution === 'ray'
  return (
    <Card>
      <CardHeader>
        <CardTitle icon={CpuIcon}>
          {role.name === 'server' ? t('kthena.topology.resourceAndImage') : role.name}
        </CardTitle>
      </CardHeader>
      <CardContent className="grid gap-5">
        <div className="grid items-start gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <ServingField
            form={form}
            name={`roles.${index}.instances`}
            readOnly={!!state.edit}
            numeric
            label={t('kthena.topology.instances')}
            help={t('kthena.topology.instancesHelp')}
          />
          {showExecution && (
            <ServingSelect
              form={form}
              name={`roles.${index}.execution`}
              label={t('kthena.topology.execution')}
              options={[
                { value: 'single', label: t('kthena.topology.single') },
                {
                  value: 'ray',
                  label: t('kthena.topology.ray'),
                  disabled: state.capabilities.isError || !rayProfile?.enabled,
                },
              ]}
              onChange={(v) => {
                form.setValue(`roles.${index}.nodesPerInstance`, v === 'ray' ? 2 : 1)
                if (v === 'single') form.setValue(`roles.${index}.worker`, undefined)
              }}
            />
          )}
          {role.execution === 'ray' && (
            <ServingField
              form={form}
              name={`roles.${index}.nodesPerInstance`}
              numeric
              label={t('kthena.topology.nodesPerInstance')}
            />
          )}
        </div>
        <PodFields state={state} index={index} kind="entry" />
        {(tensorParallel > 1 || role.execution === 'ray') && (
          <p className="text-muted-foreground text-sm">
            <span className="text-foreground mr-2 font-mono">
              TP / PP {tensorParallel || '—'} / {pipelineParallel || '—'}
            </span>
            {t('kthena.topology.automaticParallel')}
          </p>
        )}
        {role.execution === 'ray' && (
          <>
            <label className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={!!role.worker}
                onCheckedChange={(v) =>
                  form.setValue(
                    `roles.${index}.worker`,
                    v ? structuredClone(role.entry) : undefined
                  )
                }
              />
              {t('kthena.topology.separateWorker')}
            </label>
            {role.worker && <PodFields state={state} index={index} kind="worker" />}
          </>
        )}
        <section className="grid gap-3 border-t pt-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="flex items-center gap-1.5">
              <h4 className="text-sm font-medium">{t('kthena.topology.engineOptions')}</h4>
              <ServingHelp label={t('kthena.topology.engineOptions')}>
                {t('kthena.topology.engineOptionsHelp')}
              </ServingHelp>
            </div>
            <Button
              variant="secondary"
              type="button"
              onClick={() => config.append({ key: '', value: '' })}
            >
              <PlusIcon className="size-4" />
              {t('kthena.topology.addOption')}
            </Button>
          </div>
          {config.fields.map((field, i) => (
            <div
              key={field.id}
              className="grid items-start gap-3 rounded-md border p-3 sm:grid-cols-[1fr_1fr_auto]"
            >
              <ServingField
                form={form}
                name={`roles.${index}.configItems.${i}.key`}
                label={t('kthena.form.fields.configKey')}
                required
              />
              <ServingField
                form={form}
                name={`roles.${index}.configItems.${i}.value`}
                label={t('kthena.form.fields.configValue')}
              />
              <Button
                variant="ghost"
                size="sm"
                type="button"
                className="self-end justify-self-end"
                onClick={() => config.remove(i)}
              >
                {t('kthena.topology.removeEntry')}
              </Button>
            </div>
          ))}
        </section>
      </CardContent>
    </Card>
  )
}
export function ServingRoleFields({ state }: { state: ServingFormState }) {
  const summary = resourceSummary(state.values)
  return (
    <div className="grid items-start gap-4">
      {state.values.roles.map((r, i) => (
        <RoleFields state={state} key={r.name} index={i} />
      ))}
      {summary && (
        <BillingPricePreview
          entries={[
            {
              resourceList: Object.fromEntries(
                Object.entries(summary.resources)
                  .filter(([, value]) => value > 0)
                  .map(([name, value]) => [
                    name,
                    name === 'memory' ? `${value / 1024 ** 3}Gi` : String(value),
                  ])
              ),
            },
          ]}
        />
      )}
    </div>
  )
}
