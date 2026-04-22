import { FieldPath, FieldValues, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'

import { ScheduleType } from '@/services/api/vcjob'

export function ScheduleTypeFormField<T extends FieldValues>({
  form,
  name,
}: {
  form: UseFormReturn<T>
  name: FieldPath<T>
}) {
  const { t } = useTranslation()

  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem className="space-y-3">
          <FormLabel>{t('scheduleTypeFormField.label')}</FormLabel>
          <FormDescription>{t('scheduleTypeFormField.description')}</FormDescription>
          <FormControl>
            <RadioGroup
              className="grid gap-3 md:grid-cols-2"
              value={String(field.value ?? ScheduleType.Normal)}
              onValueChange={(value) => field.onChange(Number(value))}
            >
              <FormItem className="flex items-center gap-3 rounded-md border p-3">
                <FormControl>
                  <RadioGroupItem value={String(ScheduleType.Normal)} />
                </FormControl>
                <FormLabel className="font-normal">{t('scheduleTypeFormField.normal')}</FormLabel>
              </FormItem>
              <FormItem className="flex items-center gap-3 rounded-md border p-3">
                <FormControl>
                  <RadioGroupItem value={String(ScheduleType.Backfill)} />
                </FormControl>
                <FormLabel className="font-normal">{t('scheduleTypeFormField.backfill')}</FormLabel>
              </FormItem>
            </RadioGroup>
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}
