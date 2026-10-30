import { Link } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

import Combobox from '@/components/form/combobox'

import { ServingField, ServingSelect } from './serving-fields'
import type { ServingFormState } from './use-serving-form'

export function ServingModelFields({ state }: { state: ServingFormState }) {
  const { form, t, values, modelQuery } = state
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('kthena.form.sections.modelSource')}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-5">
        <ServingField
          form={form}
          name="name"
          readOnly={!!state.edit}
          label={t('kthena.form.fields.serviceName')}
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
        {values.modelSource === 'platform' ? (
          <>
            {modelQuery.isPending ? (
              <p role="status">{t('kthena.topology.loading')}</p>
            ) : modelQuery.isError ? (
              <div role="alert">
                {t('kthena.topology.modelError')}
                <Button type="button" variant="outline" onClick={() => void modelQuery.refetch()}>
                  {t('kthena.topology.retry')}
                </Button>
              </div>
            ) : (
              <FormField
                control={form.control}
                name="platformModelId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('kthena.form.fields.platformModel')}</FormLabel>
                    <FormControl>
                      <Combobox
                        items={(modelQuery.data ?? []).map((m) => ({
                          value: String(m.id),
                          label: m.name,
                        }))}
                        current={String(field.value ?? '')}
                        handleSelect={(v) => field.onChange(Number(v))}
                        formTitle={t('kthena.form.fields.platformModel')}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
          </>
        ) : (
          <>
            <ServingField form={form} name="modelURI" label={t('kthena.form.fields.modelURI')} />
            <ServingField form={form} name="modelRevision" label={t('kthena.topology.revision')} />
            <p className="text-muted-foreground text-sm">{t('kthena.topology.uri')}</p>
            <Link
              to="/portal/data/models"
              search={{ organization: undefined }}
              className="text-primary text-sm underline"
            >
              {t('kthena.topology.saveModel')}
            </Link>
          </>
        )}
        <ServingField
          form={form}
          name="servedModel"
          readOnly={!!state.edit}
          label={t('kthena.form.fields.servedModel')}
        />
        <p className="text-muted-foreground text-sm">{t('kthena.topology.routeModel')}</p>
      </CardContent>
    </Card>
  )
}
