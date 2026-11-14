import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

import type { ServingPodProfile, ServingRole } from '@/services/api/inference'

function PodProfileSummary({ profile, label }: { profile: ServingPodProfile; label?: string }) {
  const { t } = useTranslation()
  const selectors = profile.selectors ?? []
  const resources = Object.entries(profile.extendedResources ?? {})

  return (
    <div className="space-y-2">
      {label && <p className="font-medium">{label}</p>}
      <dl className="grid gap-2">
        <div>
          <dt className="text-muted-foreground">{t('kthena.topology.imageLabel')}</dt>
          <dd className="mt-1 font-mono text-xs break-all">{profile.image}</dd>
        </div>
        {selectors.length > 0 && (
          <div>
            <dt className="text-muted-foreground">{t('kthena.topology.nodes')}</dt>
            <dd className="mt-1">
              <ul className="grid gap-1">
                {selectors.map(({ key, operator, values }, index) => {
                  const hostname =
                    key === 'kubernetes.io/hostname' && (operator === 'In' || operator === 'NotIn')
                  return (
                    <li key={`${key}-${index}`} className="flex flex-wrap items-baseline gap-x-2">
                      {hostname ? (
                        <span>
                          {t(
                            operator === 'In'
                              ? 'kthena.topology.include'
                              : 'kthena.topology.exclude'
                          )}
                        </span>
                      ) : (
                        <>
                          <span className="font-mono break-all">{key}</span>
                          <span>{operator}</span>
                        </>
                      )}
                      {values?.length ? (
                        <span className="font-mono [overflow-wrap:anywhere]">
                          {values.join(', ')}
                        </span>
                      ) : null}
                    </li>
                  )
                })}
              </ul>
            </dd>
          </div>
        )}
        {resources.length > 0 && (
          <div>
            <dt className="text-muted-foreground">
              {t('kthena.topology.extendedResourcesPerPod')}
            </dt>
            <dd className="mt-1 flex flex-wrap gap-1.5">
              {resources.map(([name, amount]) => (
                <Badge
                  key={name}
                  variant="secondary"
                  className="max-w-full font-mono [overflow-wrap:anywhere] whitespace-normal"
                >
                  {name}: {amount}
                </Badge>
              ))}
            </dd>
          </div>
        )}
      </dl>
    </div>
  )
}

export function ServingRoleSummary({ roles }: { roles: ServingRole[] }) {
  const { t } = useTranslation()
  return (
    <div className="grid [grid-template-columns:repeat(auto-fit,minmax(min(100%,22rem),1fr))] items-start gap-3">
      {roles.map((role) => (
        <div key={role.name} className="min-w-0 space-y-3 rounded-md border p-3 text-sm">
          <div className="space-y-1">
            <p className="font-medium">
              {role.name} ·{' '}
              {t(role.execution === 'ray' ? 'kthena.topology.ray' : 'kthena.topology.single')}
            </p>
            <p className="text-muted-foreground">
              {role.instances} × {role.nodesPerInstance} Pods · TP {role.tensorParallel} / PP{' '}
              {role.pipelineParallel}
            </p>
          </div>
          <PodProfileSummary
            profile={role.entry}
            label={role.worker ? t('kthena.topology.entry') : undefined}
          />
          {role.worker && (
            <div className="border-t pt-3">
              <PodProfileSummary profile={role.worker} label={t('kthena.topology.worker')} />
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
