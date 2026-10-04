import { BookmarkPlusIcon, Loader2Icon, PencilIcon, RocketIcon, Trash2Icon } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

import { deploymentPresets } from './serving-form-schema'
import type { ServingFormState } from './use-serving-form'

export function ServingPresets({ state }: { state: ServingFormState }) {
  const {
    t,
    activePresetKey,
    setTemplateToDelete,
    userTemplates,
    isUserTemplatesLoading,
    applyPreset,
    applyUserTemplate,
    openTemplateDialog,
    activeUserTemplate,
  } = state
  return (
    <section className="space-y-2" aria-labelledby="runtime-template-heading">
      <div className="flex flex-wrap items-center justify-between gap-2 px-1">
        <div className="flex items-center gap-2">
          <span className="bg-primary/10 text-primary flex size-7 items-center justify-center rounded-md">
            <RocketIcon className="size-4" />
          </span>
          <h2 id="runtime-template-heading" className="text-sm font-semibold">
            {t('kthena.form.sections.presets')}
          </h2>
        </div>
        <div className="flex items-center gap-1.5">
          {activeUserTemplate && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-8"
              onClick={() => openTemplateDialog(activeUserTemplate)}
            >
              <PencilIcon className="size-3.5" />
              {t('kthena.form.template.update')}
            </Button>
          )}
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-8"
            onClick={() => openTemplateDialog()}
          >
            <BookmarkPlusIcon className="size-3.5" />
            {t('kthena.form.template.save')}
          </Button>
        </div>
      </div>

      <div
        role="radiogroup"
        aria-label={t('kthena.form.sections.presets')}
        className="bg-muted/25 grid gap-1.5 rounded-xl border p-1.5 sm:grid-cols-2"
      >
        {deploymentPresets.map((preset) => {
          const Icon = preset.icon
          const selected = activePresetKey === preset.key
          return (
            <button
              key={preset.key}
              type="button"
              role="radio"
              aria-checked={selected}
              onClick={() => applyPreset(preset)}
              className={[
                'flex min-h-20 items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-all',
                selected
                  ? 'border-primary/30 bg-background ring-primary/15 shadow-sm ring-1'
                  : 'hover:bg-background/80 border-transparent',
              ].join(' ')}
            >
              <div className="bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-md">
                <Icon className="size-4" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="truncate text-sm font-medium">{preset.title}</span>
                  {selected && (
                    <Badge className="h-5 px-1.5 text-[10px]">
                      {t('kthena.form.summary.applied')}
                    </Badge>
                  )}
                </div>
                <p className="text-muted-foreground mt-0.5 text-xs leading-4">
                  {preset.description}
                </p>
              </div>
              <div className="flex shrink-0 gap-1">
                <Badge variant="secondary">{preset.values.backendType}</Badge>
                <Badge variant="secondary">GPU</Badge>
              </div>
            </button>
          )
        })}
      </div>

      {(isUserTemplatesLoading || userTemplates.length > 0) && (
        <div className="space-y-1.5 pt-1">
          <div className="text-muted-foreground flex items-center gap-2 px-1 text-xs font-medium">
            {t('kthena.form.template.mine')}
            {isUserTemplatesLoading ? (
              <Loader2Icon className="size-3 animate-spin" />
            ) : (
              <Badge variant="secondary" className="h-5 px-1.5 text-[10px]">
                {userTemplates.length}
              </Badge>
            )}
          </div>
          {!isUserTemplatesLoading && (
            <div
              role="radiogroup"
              aria-label={t('kthena.form.template.mine')}
              className="bg-muted/25 grid gap-1.5 rounded-xl border p-1.5 sm:grid-cols-2"
            >
              {userTemplates.map((template) => {
                const selected = activePresetKey === `user:${template.id}`
                const isGPU = template.config.roles?.some((role) => Number(role.entry.gpu) > 0)
                return (
                  <div
                    key={template.id}
                    className={[
                      'flex min-w-0 items-stretch overflow-hidden rounded-lg border transition-all',
                      selected
                        ? 'border-primary/30 bg-background ring-primary/15 shadow-sm ring-1'
                        : 'hover:bg-background/80 border-transparent',
                    ].join(' ')}
                  >
                    <button
                      type="button"
                      role="radio"
                      aria-checked={selected}
                      className="flex min-w-0 flex-1 items-center gap-3 px-3 py-2.5 text-left"
                      onClick={() => applyUserTemplate(template)}
                    >
                      <div className="bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-md">
                        <BookmarkPlusIcon className="size-4" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <span className="truncate text-sm font-medium">{template.name}</span>
                          {selected && (
                            <Badge className="h-5 px-1.5 text-[10px]">
                              {t('kthena.form.summary.applied')}
                            </Badge>
                          )}
                        </div>
                        <p className="text-muted-foreground mt-0.5 truncate text-xs leading-4">
                          {template.description || t('kthena.form.template.noDescription')}
                        </p>
                      </div>
                      <div className="flex shrink-0 gap-1">
                        <Badge variant="secondary">{template.config.backendType}</Badge>
                        <Badge variant="secondary">{isGPU ? 'GPU' : 'CPU'}</Badge>
                      </div>
                    </button>
                    <div className="bg-muted/20 flex shrink-0 flex-col justify-center border-l p-1">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-7"
                        aria-label={t('kthena.form.template.update')}
                        title={t('kthena.form.template.update')}
                        onClick={() => openTemplateDialog(template)}
                      >
                        <PencilIcon className="size-3.5" />
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="text-muted-foreground hover:text-destructive size-7"
                        aria-label={t('kthena.form.template.delete')}
                        title={t('kthena.form.template.delete')}
                        onClick={() => setTemplateToDelete(template)}
                      >
                        <Trash2Icon className="size-3.5" />
                      </Button>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      )}
    </section>
  )
}
