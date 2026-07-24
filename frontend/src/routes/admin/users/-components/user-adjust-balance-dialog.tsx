import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import { AdjustUserExtraBalanceReq, apiAdminAdjustUserExtraBalance } from '@/services/api/billing'

import { AdminUserRow } from './types'

const formSchema = z.object({
  delta: z.number(),
  reason: z.string().optional(),
})

type FormValues = z.infer<typeof formSchema>

interface UserAdjustBalanceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: AdminUserRow | null
}

export function UserAdjustBalanceDialog({
  open,
  onOpenChange,
  user,
}: UserAdjustBalanceDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { delta: 0, reason: '' },
  })

  useEffect(() => {
    if (open) form.reset({ delta: 0, reason: '' })
  }, [form, open])

  const { mutate: adjustBalance, isPending } = useMutation({
    mutationFn: (values: AdjustUserExtraBalanceReq) => {
      if (!user) throw new Error('No user selected')
      return apiAdminAdjustUserExtraBalance(user.name, values)
    },
    onSuccess: (_, values) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] })
      queryClient.invalidateQueries({ queryKey: ['admin', 'users', 'billing-summary'] })
      toast.success(
        `${t('userTable.adjustBalance.success', { defaultValue: '点数调整成功' })} (${values.delta > 0 ? '+' : ''}${values.delta})`
      )
      onOpenChange(false)
    },
    onError: () =>
      toast.error(t('userTable.adjustBalance.error', { defaultValue: '点数调整失败' })),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {t('userTable.adjustBalance.title', { defaultValue: '调整额外点数' })}
          </DialogTitle>
          <DialogDescription>
            {t('userTable.adjustBalance.description', {
              defaultValue: '按增量调整用户 extra 点数，可输入正数或负数。',
            })}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit((values) => adjustBalance(values))}
            className="space-y-4"
          >
            <FormField
              control={form.control}
              name="delta"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('userTable.adjustBalance.delta', { defaultValue: '变更值' })}
                  </FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      value={field.value ?? 0}
                      onChange={(event) => field.onChange(Number(event.target.value))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="reason"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('userTable.adjustBalance.reason', { defaultValue: '原因' })}
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('userTable.adjustBalance.reasonPlaceholder', {
                        defaultValue: '可选，便于审计',
                      })}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="submit" disabled={isPending}>
                {isPending ? t('common.saving') : t('common.saveChanges')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
