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

import { IUserAttributes, apiAdminUpdateUserAttributes } from '@/services/api/admin/user'

import { AdminUserRow } from './types'

const userFormSchema = (t: (key: string) => string) =>
  z.object({
    nickname: z.string().optional(),
    email: z.string().email(t('userForm.emailError')).optional().or(z.literal('')),
    teacher: z.string().optional().or(z.literal('')),
    group: z.string().optional().or(z.literal('')),
    phone: z.string().optional().or(z.literal('')),
  })

type UserFormValues = z.infer<ReturnType<typeof userFormSchema>>

interface UserEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: AdminUserRow | null
}

export function UserEditDialog({ open, onOpenChange, user }: UserEditDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const form = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema(t)),
    defaultValues: {
      nickname: '',
      email: '',
      teacher: '',
      group: '',
      phone: '',
    },
  })

  useEffect(() => {
    if (user) {
      form.reset({
        nickname: user.attributes.nickname || '',
        email: user.attributes.email || '',
        teacher: user.attributes.teacher || '',
        group: user.attributes.group || '',
        phone: user.attributes.phone || '',
      })
    }
  }, [form, user])

  const { mutate: updateUser, isPending } = useMutation({
    mutationFn: (values: UserFormValues) => {
      if (!user) throw new Error('No user selected')
      const updateData: IUserAttributes = { ...user.attributes, ...values }
      return apiAdminUpdateUserAttributes(user.name, updateData)
    },
    onSuccess: () => {
      toast.success(t('userEditDialog.successToast'))
      queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] })
      onOpenChange(false)
    },
    onError: () => toast.error(t('userEditDialog.errorToast')),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{t('userEditDialog.title')}</DialogTitle>
          <DialogDescription>
            {t('userEditDialog.description', { name: user?.name })}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit((values) => updateUser(values))} className="space-y-4">
            {(['nickname', 'email', 'teacher', 'group', 'phone'] as const).map((name) => (
              <FormField
                key={name}
                control={form.control}
                name={name}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t(`userEditDialog.${name}Label`)}</FormLabel>
                    <FormControl>
                      <Input placeholder={t(`userEditDialog.${name}Placeholder`)} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ))}
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
