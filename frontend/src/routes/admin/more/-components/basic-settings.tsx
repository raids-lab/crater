import { zodResolver } from '@hookform/resolvers/zod'
import { useAtom } from 'jotai'
import { FileCogIcon } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import {
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Form, FormControl, FormField, FormItem, FormMessage } from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'

import { globalSettings } from '@/utils/store'

const basicSettingsSchema = z.object({
  scheduler: z.enum(['volcano', 'colocate', 'sparse'], {
    invalid_type_error: 'invalidType',
    required_error: 'required',
  }),
  hideUsername: z.boolean().default(false),
})

type BasicFormSchema = z.infer<typeof basicSettingsSchema>

export function BasicSettings() {
  const { t } = useTranslation()
  const [settings, setSettings] = useAtom(globalSettings)

  const form = useForm<BasicFormSchema>({
    resolver: zodResolver(basicSettingsSchema),
    defaultValues: settings,
  })

  const onSubmit = () => {
    toast.success(t('systemSetting.toast.success'))
    setSettings(form.getValues())
    window.location.reload()
  }

  return (
    <>
      {/* 调度器设置 */}
      <CardHeader>
        <CardTitle>{t('systemSetting.scheduler.title')}</CardTitle>
        <CardDescription>{t('systemSetting.scheduler.description')}</CardDescription>
      </CardHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent>
            <FormField
              control={form.control}
              name="scheduler"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Select onValueChange={field.onChange} defaultValue={field.value}>
                      <SelectTrigger>
                        <SelectValue placeholder={t('systemSetting.scheduler.placeholder')} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="volcano">
                          {t('systemSetting.scheduler.volcano')}
                        </SelectItem>
                        <SelectItem value="colocate">
                          {t('systemSetting.scheduler.colocate')}
                        </SelectItem>
                        <SelectItem value="sparse">
                          {t('systemSetting.scheduler.sparse')}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
          <CardFooter className="bg-muted/10 px-6 py-4">
            <Button type="submit">
              <FileCogIcon className="mr-2 h-4 w-4" />
              {t('systemSetting.scheduler.submit')}
            </Button>
          </CardFooter>
        </form>
      </Form>

      <Separator />

      {/* 用户名显示设置 */}
      <CardHeader>
        <CardTitle>{t('systemSetting.username.title')}</CardTitle>
        <CardDescription>{t('systemSetting.username.description')}</CardDescription>
      </CardHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent>
            <FormField
              control={form.control}
              name="hideUsername"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Select
                      onValueChange={(value) => {
                        field.onChange(value === 'true')
                      }}
                      defaultValue={field.value ? 'true' : 'false'}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder={t('systemSetting.username.placeholder')} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="true">{t('systemSetting.username.yes')}</SelectItem>
                        <SelectItem value="false">{t('systemSetting.username.no')}</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
          <CardFooter className="bg-muted/10 px-6 py-4">
            <Button type="submit">
              <FileCogIcon className="mr-2 h-4 w-4" />
              {t('systemSetting.username.submit')}
            </Button>
          </CardFooter>
        </form>
      </Form>
    </>
  )
}
