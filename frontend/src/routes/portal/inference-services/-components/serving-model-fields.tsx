import { Link } from '@tanstack/react-router'
import { BoxIcon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'

import Combobox from '@/components/form/combobox'
import FormLabelMust from '@/components/form/form-label-must'
import CardTitle from '@/components/label/card-title'

import { ServingField, ServingSelect } from './serving-fields'
import type { ServingFormState } from './use-serving-form'

export function ServingModelFields({ state }: { state: ServingFormState }) {
  const { form, t, values, modelQuery } = state
  return (
    <Card>
      <CardHeader>
        <CardTitle icon={BoxIcon}>{t('kthena.form.sections.modelSource')}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-5">
        <div className="grid items-start gap-4 sm:grid-cols-2">
          <ServingField
            form={form}
            name="name"
            readOnly={!!state.edit}
            required
            label={t('kthena.form.fields.serviceName')}
            description={state.edit ? undefined : t('kthena.form.nameHelp')}
            placeholder="qwen-small"
          />
          <ServingSelect
            form={form}
            name="modelSource"
            label={t('kthena.form.fields.source')}
            options={['platform', 'external'].map((value) => ({
              value,
              label: t(`kthena.form.source.${value}`),
            }))}
          />
        </div>
        {values.modelSource === 'platform' ? (
          <>
            {modelQuery.isPending ? (
              <p role="status" className="text-muted-foreground text-sm">
                {t('kthena.topology.loading')}
              </p>
            ) : modelQuery.isError ? (
              <div
                role="alert"
                className="bg-muted/40 flex flex-wrap items-center justify-between gap-3 rounded-md border p-3 text-sm"
              >
                <span>{t('kthena.topology.modelError')}</span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => void modelQuery.refetch()}
                >
                  {t('kthena.topology.retry')}
                </Button>
              </div>
            ) : (modelQuery.data?.length ?? 0) === 0 ? (
              <div className="bg-muted/40 grid gap-2 rounded-md border p-3 text-sm">
                <p>{t('kthena.form.modelEmpty')}</p>
                <Link
                  to="/portal/data/models"
                  search={{ organization: undefined }}
                  className="text-primary w-fit underline underline-offset-4"
                >
                  {t('kthena.topology.saveModel')}
                </Link>
              </div>
            ) : (
              <FormField
                control={form.control}
                name="platformModelId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('kthena.form.fields.platformModel')}
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Combobox
                        items={(modelQuery.data ?? []).map((m) => ({
                          value: String(m.id),
                          label: m.name,
                        }))}
                        current={String(field.value ?? '')}
                        handleSelect={(v) => field.onChange(v ? Number(v) : undefined)}
                        formTitle={t('kthena.form.fields.platformModel')}
                      />
                    </FormControl>
                    <FormDescription>{t('kthena.form.descriptions.platformModel')}</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
          </>
        ) : (
          <>
            <div className="grid items-start gap-4 sm:grid-cols-[2fr_1fr]">
              <ServingField
                form={form}
                name="modelURI"
                required
                label={t('kthena.form.fields.modelURI')}
                placeholder="hf://Qwen/Qwen2.5-0.5B-Instruct"
                description={t('kthena.topology.uri')}
              />
              <ServingField
                form={form}
                name="modelRevision"
                label={t('kthena.topology.revision')}
              />
            </div>
            <Link
              to="/portal/data/models"
              search={{ organization: undefined }}
              className="text-primary text-sm underline"
            >
              {t('kthena.topology.saveModel')}
            </Link>
          </>
        )}
      </CardContent>
    </Card>
  )
}
