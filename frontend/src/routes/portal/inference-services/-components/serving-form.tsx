import { Loader2Icon, RocketIcon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Form } from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import FormExportButton from '@/components/form/form-export-button'
import FormImportButton from '@/components/form/form-import-button'
import PageTitle from '@/components/layout/page-title'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui-custom/alert-dialog'

import { formSchema } from './serving-form-schema'
import { ServingModelFields } from './serving-model-fields'
import { ServingPresets } from './serving-presets'
import { ServingRoleFields } from './serving-role-fields'
import { ServingSummary } from './serving-summary'
import { ServingPreflightSummary, ServingTopologyFields } from './serving-topology'
import { useServingForm } from './use-serving-form'

export function ServingForm() {
  const state = useServingForm()
  const {
    t,
    clone,
    pendingCreate,
    setPendingCreate,
    templateDialogOpen,
    setTemplateDialogOpen,
    editingTemplate,
    setEditingTemplate,
    templateToDelete,
    setTemplateToDelete,
    templateName,
    setTemplateName,
    templateDescription,
    setTemplateDescription,
    form,
    createService,
    isPending,
    isSavingUserTemplate,
    deleteUserTemplate,
    isDeletingUserTemplate,
    onSubmit,
    submitUserTemplate,
  } = state
  const validation = formSchema.safeParse(state.values)
  const showErrors = Object.keys(form.formState.errors).length > 0
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-4">
        <PageTitle
          title={t('kthena.form.title')}
          description={
            clone
              ? t('kthena.form.cloneDescription', { name: clone })
              : t('kthena.form.description')
          }
        >
          <FormImportButton
            metadata={{ type: 'serving', version: '2' }}
            form={form}
            dataProcessor={(data) => formSchema.parse(data)}
          />
          <FormExportButton metadata={{ type: 'serving', version: '2' }} form={form} />
        </PageTitle>

        {state.edit ? (
          <p className="text-muted-foreground text-sm">{t('kthena.topology.editHelp')}</p>
        ) : (
          <ServingPresets state={state} />
        )}

        <nav aria-label={t('kthena.topology.steps')} className="flex flex-wrap gap-2">
          {['model', 'topology', 'roles', 'preview'].map((step, index) => (
            <Button
              key={step}
              type="button"
              variant={state.step === index ? 'default' : 'outline'}
              disabled={index > state.step}
              onClick={() => state.setStep(index)}
              aria-current={state.step === index ? 'step' : undefined}
            >
              {index + 1}. {t(`kthena.topology.step.${step}`)}
            </Button>
          ))}
        </nav>
        {showErrors && !validation.success && (
          <ul role="alert" className="text-destructive list-inside list-disc text-sm">
            {validation.error.issues.map((issue, i) => (
              <li key={i}>
                {issue.path.join('.')}：{issue.message}
              </li>
            ))}
          </ul>
        )}
        <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
          <div className="min-w-0">
            {state.step === 0 && <ServingModelFields state={state} />}
            {state.step === 1 && <ServingTopologyFields state={state} />}
            {state.step === 2 && <ServingRoleFields state={state} />}
            {state.step === 3 && <ServingPreflightSummary state={state} />}
          </div>
          <ServingSummary state={state} />
        </div>
        <div className="bg-background sticky bottom-0 z-10 flex justify-end gap-3 border-t py-4">
          {state.step > 0 && (
            <Button variant="outline" type="button" onClick={() => state.setStep(state.step - 1)}>
              {t('kthena.topology.previous')}
            </Button>
          )}
          {state.step < 3 ? (
            <Button
              type="button"
              disabled={state.preflight.isPending}
              onClick={() => void state.next()}
            >
              {t(state.step === 2 ? 'kthena.topology.check' : 'kthena.topology.next')}
            </Button>
          ) : (
            <Button type="submit" disabled={isPending || state.preflight.isPending}>
              <RocketIcon className="size-4" />
              {t(state.previewCurrent ? 'kthena.form.submit' : 'kthena.topology.check')}
            </Button>
          )}
        </div>
      </form>

      <AlertDialog
        open={!!pendingCreate}
        onOpenChange={(open) => {
          if (!open && !isPending) setPendingCreate(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('kthena.form.confirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingCreate?.name} · {state.preview?.queue} · {state.preview?.pods} Pods
              <span className="block break-all">
                {Object.entries(state.preview?.totalResources ?? {})
                  .map(([key, value]) => `${key}: ${value}`)
                  .join(' · ')}
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isPending}>{t('kthena.actions.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              disabled={isPending}
              onClick={(event) => {
                event.preventDefault()
                if (pendingCreate) createService(pendingCreate)
              }}
            >
              {t('kthena.actions.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <Dialog
        open={templateDialogOpen}
        onOpenChange={(open) => {
          setTemplateDialogOpen(open)
          if (!open) setEditingTemplate(null)
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>
              {editingTemplate
                ? t('kthena.form.template.updateTitle')
                : t('kthena.form.template.saveTitle')}
            </DialogTitle>
            <DialogDescription>{t('kthena.form.template.dialogDescription')}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-1">
            <div className="grid gap-2">
              <label htmlFor="kthena-template-name" className="text-sm font-medium">
                {t('kthena.form.template.name')}
              </label>
              <Input
                id="kthena-template-name"
                value={templateName}
                maxLength={64}
                autoFocus
                onChange={(event) => setTemplateName(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <label htmlFor="kthena-template-description" className="text-sm font-medium">
                {t('kthena.form.template.description')}
              </label>
              <Input
                id="kthena-template-description"
                value={templateDescription}
                maxLength={512}
                onChange={(event) => setTemplateDescription(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={isSavingUserTemplate}
              onClick={() => setTemplateDialogOpen(false)}
            >
              {t('kthena.actions.cancel')}
            </Button>
            <Button type="button" disabled={isSavingUserTemplate} onClick={submitUserTemplate}>
              {isSavingUserTemplate && <Loader2Icon className="size-4 animate-spin" />}
              {editingTemplate ? t('kthena.form.template.update') : t('kthena.form.template.save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(templateToDelete)}
        onOpenChange={(open) => {
          if (!open) setTemplateToDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('kthena.form.template.deleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('kthena.form.template.deleteDescription', { name: templateToDelete?.name ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeletingUserTemplate}>
              {t('kthena.actions.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={isDeletingUserTemplate || !templateToDelete}
              onClick={() => {
                if (templateToDelete) deleteUserTemplate(templateToDelete.id)
              }}
            >
              {isDeletingUserTemplate && <Loader2Icon className="size-4 animate-spin" />}
              {t('kthena.form.template.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Form>
  )
}
