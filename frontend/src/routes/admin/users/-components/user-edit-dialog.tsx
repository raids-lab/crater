import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
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
import { Switch } from '@/components/ui/switch'

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

import {
  IAdminUpdateUserAttributesReq,
  apiAdminUpdateUserAttributes,
} from '@/services/api/admin/user'

import { showErrorToast } from '@/utils/toast'

import { AdminUserRow } from './types'

const userFormSchema = (t: (key: string) => string) =>
  z.object({
    containerSysAdminEnabled: z.boolean(),
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
  const [pendingValues, setPendingValues] = useState<UserFormValues | null>(null)
  const form = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema(t)),
    defaultValues: {
      containerSysAdminEnabled: false,
      nickname: '',
      email: '',
      teacher: '',
      group: '',
      phone: '',
    },
  })

  useEffect(() => {
    if (open && user) {
      form.reset({
        containerSysAdminEnabled: user.containerSysAdminEnabled ?? false,
        nickname: user.attributes.nickname || '',
        email: user.attributes.email || '',
        teacher: user.attributes.teacher || '',
        group: user.attributes.group || '',
        phone: user.attributes.phone || '',
      })
    }
    if (!open) setPendingValues(null)
  }, [form, user, open])

  const { mutate: updateUser, isPending } = useMutation({
    mutationFn: (values: UserFormValues) => {
      if (!user) throw new Error('No user selected')
      const updateData: IAdminUpdateUserAttributesReq = { ...user.attributes, ...values }
      return apiAdminUpdateUserAttributes(user.name, updateData)
    },
    onSuccess: () => {
      toast.success(t('userEditDialog.successToast'))
      queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] })
      onOpenChange(false)
    },
    onError: showErrorToast,
  })

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(nextOpen) => {
          if (!isPending) onOpenChange(nextOpen)
        }}
      >
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>{t('userEditDialog.title')}</DialogTitle>
            <DialogDescription>
              {t('userEditDialog.description', { name: user?.name })}
            </DialogDescription>
          </DialogHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(setPendingValues)} className="space-y-4">
              <fieldset disabled={isPending} className="space-y-4">
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
                <FormField
                  control={form.control}
                  name="containerSysAdminEnabled"
                  render={({ field }) => (
                    <FormItem className="flex items-center justify-between gap-4">
                      <FormLabel>{t('userEditDialog.containerSysAdminEnabled')}</FormLabel>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                          disabled={isPending}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </fieldset>
              <DialogFooter>
                <Button type="submit" disabled={isPending}>
                  {isPending ? t('common.saving') : t('common.saveChanges')}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
      <AlertDialog
        open={pendingValues !== null}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setPendingValues(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('userEditDialog.confirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('userEditDialog.confirmDescription', { name: user?.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isPending}>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              disabled={isPending}
              onClick={() => {
                if (pendingValues) updateUser(pendingValues)
              }}
            >
              {t('common.saveChanges')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
