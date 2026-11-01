import { FieldPath, UseFormReturn } from 'react-hook-form'

import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import type { FormSchema } from './serving-form-schema'

export function ServingField({
  form,
  name,
  label,
  numeric = false,
  readOnly = false,
}: {
  form: UseFormReturn<FormSchema>
  name: FieldPath<FormSchema>
  label: string
  numeric?: boolean
  readOnly?: boolean
}) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            <Input
              {...field}
              readOnly={readOnly}
              value={
                typeof field.value === 'string' || typeof field.value === 'number'
                  ? field.value
                  : ''
              }
              type={numeric ? 'number' : 'text'}
              min={numeric ? 1 : undefined}
              onChange={(e) => field.onChange(numeric ? Number(e.target.value) : e.target.value)}
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}
export function ServingSelect({
  form,
  name,
  label,
  options,
  onChange,
  disabled = false,
}: {
  form: UseFormReturn<FormSchema>
  name: FieldPath<FormSchema>
  label: string
  options: Array<{ value: string; label: string; disabled?: boolean }>
  disabled?: boolean
  onChange?: (value: string) => void
}) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>{label}</FormLabel>
          <Select
            disabled={disabled}
            value={String(field.value ?? '')}
            onValueChange={(v) => {
              field.onChange(v)
              onChange?.(v)
            }}
          >
            <FormControl>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
            </FormControl>
            <SelectContent>
              {options.map((o) => (
                <SelectItem key={o.value} value={o.value} disabled={o.disabled}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}
