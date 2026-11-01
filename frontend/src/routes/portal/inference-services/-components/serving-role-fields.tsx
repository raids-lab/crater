import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useFieldArray } from 'react-hook-form'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'

import { EnvFormCard } from '@/components/form/env-form-field'
import { ImageFormField } from '@/components/form/image-form-field'
import { ResourceFormFields } from '@/components/form/resource-form-field'

import { apiResourceList, apiResourceNetworks } from '@/services/api/resource'
import { JobType } from '@/services/api/vcjob'

import { ServingField, ServingSelect } from './serving-fields'
import { ServingProfileConstraints } from './serving-profile-constraints'
import { restoreNetworkProfile } from './serving-serialization'
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
  const [envOpen, setEnvOpen] = useState(false)
  const secrets = useFieldArray({ control: form.control, name: `${prefix}.secretEnvs` })
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
  const selectedImage =
    profile.imageSource === 'platform' ? profile.platformImage.imageLink : profile.image

  return (
    <div className="grid gap-5 rounded-lg border p-4">
      <h4 className="font-medium">
        {kind === 'entry' ? t('kthena.topology.entry') : t('kthena.topology.worker')}
      </h4>
      <ServingSelect
        form={form}
        name={`${prefix}.imageSource`}
        label={t('kthena.topology.imageLabel')}
        options={['manual', 'platform'].map((value) => ({
          value,
          label: t(`kthena.topology.image.${value}`),
        }))}
      />
      {profile.imageSource === 'platform' ? (
        <ImageFormField form={form} name={`${prefix}.platformImage`} jobType={JobType.Custom} />
      ) : (
        <ServingField
          form={form}
          name={`${prefix}.image`}
          label={t('kthena.topology.imageLabel')}
        />
      )}
      {capability && (
        <div className="text-muted-foreground text-sm break-all">
          <p>
            {t('kthena.topology.approvedImages')}: {capability.images.join(', ')}
          </p>
          {capability.connector && <p>KV: {capability.connector}</p>}
          {!capability.images.includes(selectedImage ?? '') && (
            <p className="text-destructive">{t('kthena.topology.unapprovedImage')}</p>
          )}
        </div>
      )}
      <ResourceFormFields
        form={form}
        allowFractionalCPUAndMemory
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
      <ServingProfileConstraints state={state} index={index} kind={kind} />
      <details>
        <summary className="cursor-pointer text-sm">{t('kthena.topology.nodes')}</summary>
        <label className="my-3 flex items-center gap-2 text-sm">
          <Checkbox
            checked={profile.nodeSelector.enable}
            onCheckedChange={(v) => form.setValue(`${prefix}.nodeSelector.enable`, v === true)}
          />
          {t('kthena.topology.constrainNodes')}
        </label>
        {profile.nodeSelector.enable && (
          <>
            <ServingSelect
              form={form}
              name={`${prefix}.nodeSelector.mode`}
              label={t('kthena.topology.nodeMode')}
              options={['include', 'exclude'].map((value) => ({
                value,
                label: t(`kthena.topology.${value}`),
              }))}
            />
            {state.nodeQuery.isError ? (
              <Button type="button" onClick={() => void state.nodeQuery.refetch()}>
                {t('kthena.topology.retry')}
              </Button>
            ) : (
              state.nodeQuery.data?.map((node) => (
                <label key={node.value} className="my-2 flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={profile.nodeSelector.nodes?.includes(node.value)}
                    onCheckedChange={(checked) =>
                      form.setValue(
                        `${prefix}.nodeSelector.nodes`,
                        checked
                          ? [...(profile.nodeSelector.nodes ?? []), node.value]
                          : (profile.nodeSelector.nodes ?? []).filter((n) => n !== node.value)
                      )
                    }
                  />
                  {node.label}
                </label>
              ))
            )}
          </>
        )}
      </details>
      <details>
        <summary>{t('kthena.topology.credentials')}</summary>
        <div className="mt-3 grid gap-3">
          {secrets.fields.map((secret, i) => (
            <div key={secret.id} className="grid gap-2 sm:grid-cols-3">
              <ServingField
                form={form}
                name={`${prefix}.secretEnvs.${i}.name`}
                label={t('kthena.topology.envName')}
              />
              <ServingField form={form} name={`${prefix}.secretEnvs.${i}.secret`} label="Secret" />
              <ServingField form={form} name={`${prefix}.secretEnvs.${i}.key`} label="Key" />
              <Button type="button" variant="ghost" onClick={() => secrets.remove(i)}>
                {t('kthena.form.template.delete')}
              </Button>
            </div>
          ))}
          <Button
            type="button"
            variant="outline"
            onClick={() => secrets.append({ name: 'HF_AUTH_TOKEN', secret: '', key: '' })}
          >
            {t('kthena.topology.addCredential')}
          </Button>
        </div>
      </details>
      <EnvFormCard
        form={form}
        envPath={`${prefix}.envs`}
        open={envOpen}
        setOpen={setEnvOpen}
        cardTitle={t('kthena.form.sections.env')}
      />
    </div>
  )
}
function RoleFields({ state, index }: { state: ServingFormState; index: number }) {
  const { form, t, values } = state
  const role = values.roles[index]
  const config = useFieldArray({ control: form.control, name: `roles.${index}.configItems` })
  const canRay = state.capabilities.data?.some(
    (c) =>
      c.engine === values.backendType &&
      c.layout === values.layout &&
      c.execution === 'ray' &&
      c.enabled
  )
  return (
    <Card>
      <CardHeader>
        <CardTitle>{role.name}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <ServingField
            form={form}
            name={`roles.${index}.instances`}
            readOnly={!!state.edit}
            numeric
            label={t('kthena.topology.instances')}
          />
          <ServingSelect
            form={form}
            name={`roles.${index}.execution`}
            label={t('kthena.topology.execution')}
            options={[
              { value: 'single', label: t('kthena.topology.single') },
              { value: 'ray', label: 'Ray', disabled: !canRay },
            ]}
            onChange={(v) => {
              form.setValue(`roles.${index}.nodesPerInstance`, v === 'ray' ? 2 : 1)
              form.setValue(`roles.${index}.pipelineParallel`, v === 'ray' ? 2 : 1)
              if (v === 'single') form.setValue(`roles.${index}.worker`, undefined)
            }}
          />
          {role.execution === 'ray' && (
            <ServingField
              form={form}
              name={`roles.${index}.nodesPerInstance`}
              numeric
              label={t('kthena.topology.nodesPerInstance')}
            />
          )}
          <ServingField
            form={form}
            name={`roles.${index}.tensorParallel`}
            numeric
            label="Tensor parallel (TP)"
          />
          <ServingField
            form={form}
            name={`roles.${index}.pipelineParallel`}
            numeric
            label="Pipeline parallel (PP)"
          />
        </div>
        <p className="text-muted-foreground text-sm">
          {t(
            role.execution === 'ray'
              ? 'kthena.topology.rayConstraint'
              : 'kthena.topology.singleConstraint'
          )}
        </p>
        <PodFields state={state} index={index} kind="entry" />
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
        <details>
          <summary className="cursor-pointer text-sm">{t('kthena.form.sections.advanced')}</summary>
          <div className="mt-4 grid gap-3">
            {config.fields.map((field, i) => (
              <div key={field.id} className="grid gap-2 sm:grid-cols-[1fr_1fr_auto]">
                <ServingField
                  form={form}
                  name={`roles.${index}.configItems.${i}.key`}
                  label={t('kthena.form.fields.configKey')}
                />
                <ServingField
                  form={form}
                  name={`roles.${index}.configItems.${i}.value`}
                  label={t('kthena.form.fields.configValue')}
                />
                <Button variant="ghost" type="button" onClick={() => config.remove(i)}>
                  {t('kthena.form.template.delete')}
                </Button>
              </div>
            ))}
            <Button
              variant="outline"
              type="button"
              onClick={() => config.append({ key: '', value: '' })}
            >
              {t('kthena.topology.addOption')}
            </Button>
          </div>
        </details>
      </CardContent>
    </Card>
  )
}
export function ServingRoleFields({ state }: { state: ServingFormState }) {
  return (
    <div className="grid items-start gap-4 2xl:grid-cols-2">
      {state.values.roles.map((r, i) => (
        <RoleFields state={state} key={r.name} index={i} />
      ))}
    </div>
  )
}
