import { CircleHelpIcon } from 'lucide-react'
import { FieldPath, UseFormReturn } from 'react-hook-form'

import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import FormLabelMust from '@/components/form/form-label-must'
import SimpleTooltip from '@/components/label/simple-tooltip'

import type { FormSchema } from './serving-form-schema'

export function ServingHelp({ label, children }: { label: string; children: string }) {
  return (
    <SimpleTooltip
      ariaLabel={label}
      tooltip={<span className="block max-w-64 leading-relaxed text-pretty">{children}</span>}
    >
      <CircleHelpIcon aria-hidden className="text-muted-foreground size-4 cursor-help" />
    </SimpleTooltip>
  )
}

function ServingLabel({
  label,
  required,
  help,
}: {
  label: string
  required: boolean
  help?: string
}) {
  return (
    <div className="flex items-center gap-1.5">
      <FormLabel>
        {label}
        {required && <FormLabelMust />}
      </FormLabel>
      {help && <ServingHelp label={label}>{help}</ServingHelp>}
    </div>
  )
}

export function ServingField({
  form,
  name,
  label,
  description,
  help,
  placeholder,
  numeric = false,
  readOnly = false,
  required = false,
}: {
  form: UseFormReturn<FormSchema>
  name: FieldPath<FormSchema>
  label: string
  description?: string
  help?: string
  placeholder?: string
  numeric?: boolean
  readOnly?: boolean
  required?: boolean
}) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <ServingLabel label={label} required={required} help={help} />
          <FormControl>
            <Input
              {...field}
              readOnly={readOnly}
              placeholder={placeholder}
              aria-required={required || undefined}
              value={
                typeof field.value === 'string' || typeof field.value === 'number'
                  ? field.value
                  : ''
              }
              type={numeric ? 'number' : 'text'}
              min={numeric ? 1 : undefined}
              onChange={(e) =>
                field.onChange(
                  numeric ? (e.target.value === '' ? null : e.target.valueAsNumber) : e.target.value
                )
              }
            />
          </FormControl>
          {description && <FormDescription>{description}</FormDescription>}
          {help && !description && <FormDescription className="sr-only">{help}</FormDescription>}
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
  help,
  options,
  value,
  placeholder,
  required = false,
  onChange,
  onValueChange,
  disabled = false,
}: {
  form: UseFormReturn<FormSchema>
  name: FieldPath<FormSchema>
  label: string
  help?: string
  options: Array<{ value: string; label: string; disabled?: boolean }>
  value?: string
  placeholder?: string
  required?: boolean
  disabled?: boolean
  onChange?: (value: string) => void
  onValueChange?: (value: string) => void
}) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <ServingLabel label={label} required={required} help={help} />
          <Select
            disabled={disabled}
            value={value ?? String(field.value ?? '')}
            onValueChange={
              onValueChange ??
              ((v) => {
                field.onChange(v)
                onChange?.(v)
              })
            }
          >
            <FormControl>
              <SelectTrigger className="w-full" aria-required={required || undefined}>
                <SelectValue placeholder={placeholder} />
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
          {help && <FormDescription className="sr-only">{help}</FormDescription>}
          <FormMessage />
        </FormItem>
      )}
    />
  )
}
