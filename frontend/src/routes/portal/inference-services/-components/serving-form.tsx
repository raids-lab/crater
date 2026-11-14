import { ArrowLeftIcon, ArrowRightIcon, Loader2Icon, RocketIcon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
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

import { cn } from '@/lib/utils'

import { ServingEnvironmentFields } from './serving-environment-fields'
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
    confirmCreate,
    isPending,
    isSavingUserTemplate,
    deleteUserTemplate,
    isDeletingUserTemplate,
    submitForm,
    submitUserTemplate,
  } = state
  const {
    step,
    furthestStep,
    goToStep,
    hasStepError,
    stepError,
    edit,
    preflight,
    sourceLoading,
    sourceError,
  } = state
  const title = t(edit ? 'kthena.form.editTitle' : 'kthena.form.title')
  const busy = isPending || preflight.isPending
  const configuringResources = step === 2
  if (sourceLoading || sourceError) {
    return (
      <div className="grid gap-4">
        <PageTitle title={title} />
        <Card>
          <CardContent className="flex flex-wrap items-center gap-3">
            {sourceError ? (
              <>
                <p role="alert" className="text-destructive text-sm">
                  {sourceError.message}
                </p>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => void state.cloneQuery.refetch()}
                >
                  {t('common.refresh')}
                </Button>
              </>
            ) : (
              <p role="status" className="text-muted-foreground flex items-center gap-2 text-sm">
                <Loader2Icon className="size-4 animate-spin" />
                {t('common.loading')}
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    )
  }
  return (
    <Form {...form}>
      <form noValidate onSubmit={submitForm} className="flex min-w-0 flex-1 flex-col gap-4">
        <PageTitle
          title={title}
          description={
            clone
              ? t('kthena.form.cloneDescription', { name: clone })
              : t('kthena.form.description')
          }
        >
          <div className="flex w-full flex-wrap items-center gap-2 sm:w-fit sm:justify-end sm:gap-3">
            {!state.edit && <ServingPresets state={state} />}
            {!edit && (
              <FormImportButton
                metadata={{ type: 'serving', version: '2' }}
                form={form}
                dataProcessor={(data) => formSchema.parse(data)}
              />
            )}
            <FormExportButton metadata={{ type: 'serving', version: '2' }} form={form} />
          </div>
        </PageTitle>

        {state.edit && (
          <p className="text-muted-foreground text-sm">{t('kthena.topology.editHelp')}</p>
        )}

        <nav aria-label={t('kthena.topology.steps')} className="border-b">
          <div className="flex gap-5 overflow-x-auto sm:gap-6">
            {['model', 'topology', 'roles', 'preview'].map((step, index) => (
              <button
                key={step}
                type="button"
                className="tabs-trigger-underline shrink-0"
                data-state={state.step === index ? 'active' : 'inactive'}
                disabled={index > furthestStep || busy}
                onClick={() => void goToStep(index)}
                aria-current={state.step === index ? 'step' : undefined}
              >
                {index + 1}. {t(`kthena.topology.step.${step}`)}
              </button>
            ))}
          </div>
        </nav>
        {hasStepError && (
          <p
            role="alert"
            className="border-destructive/25 bg-destructive/5 text-destructive rounded-md border px-3 py-2 text-sm"
          >
            {stepError || t('kthena.form.checkFields')}
          </p>
        )}
        <div
          className={cn(
            'grid items-start gap-4 md:gap-6',
            configuringResources
              ? 'lg:grid-cols-3'
              : 'lg:grid-cols-[minmax(0,1fr)_minmax(15rem,20rem)]'
          )}
        >
          <div className={cn('min-w-0', configuringResources && 'lg:col-span-2')}>
            {step === 0 && <ServingModelFields state={state} />}
            {step === 1 && <ServingTopologyFields state={state} />}
            {configuringResources && <ServingRoleFields state={state} />}
            {step === 3 && <ServingPreflightSummary state={state} />}
          </div>
          <div
            className={cn(
              'grid min-w-0 gap-4 md:gap-6',
              !configuringResources && 'lg:sticky lg:top-4'
            )}
          >
            <ServingSummary state={state} />
            {configuringResources && <ServingEnvironmentFields state={state} />}
          </div>
        </div>
        <div className="bg-background sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-3 border-t py-4">
          <p className="text-muted-foreground text-sm tabular-nums">{step + 1} / 4</p>
          <div className="flex gap-2">
            {step > 0 && (
              <Button
                type="button"
                variant="outline"
                disabled={busy}
                onClick={() => void goToStep(step - 1)}
              >
                <ArrowLeftIcon className="size-4" />
                {t('kthena.topology.previous')}
              </Button>
            )}
            <Button type="submit" disabled={busy}>
              {preflight.isPending || isPending ? (
                <Loader2Icon className="size-4 animate-spin" />
              ) : step === 3 && state.previewCurrent && !edit ? (
                <RocketIcon className="size-4" />
              ) : null}
              {step < 2
                ? t('kthena.topology.next')
                : step === 2 || !state.previewCurrent
                  ? t('kthena.topology.check')
                  : t(edit ? 'common.saveChanges' : 'kthena.form.submit')}
              {step < 2 && <ArrowRightIcon className="size-4" />}
            </Button>
          </div>
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
            <AlertDialogTitle>
              {t(edit ? 'kthena.form.editConfirm' : 'kthena.form.confirmTitle')}
            </AlertDialogTitle>
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
                confirmCreate()
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
