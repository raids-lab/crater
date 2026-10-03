import { zodResolver } from '@hookform/resolvers/zod'
import { useAtom } from 'jotai'
import { EyeOffIcon, FileCogIcon, WorkflowIcon } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Form, FormControl, FormField, FormItem, FormMessage } from '@/components/ui/form'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

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
    <Form {...form}>
      <div className="grid gap-4 lg:grid-cols-2">
        <form onSubmit={form.handleSubmit(onSubmit)} className="flex">
          <Card className="w-full gap-4">
            <CardHeader>
              <div className="flex items-center gap-2">
                <WorkflowIcon className="text-primary size-5" />
                <CardTitle>{t('systemSetting.scheduler.title')}</CardTitle>
              </div>
              <CardDescription>{t('systemSetting.scheduler.description')}</CardDescription>
            </CardHeader>
            <CardContent>
              <FormField
                control={form.control}
                name="scheduler"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Select onValueChange={field.onChange} value={field.value}>
                        <SelectTrigger className="w-full">
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
            <CardFooter className="bg-muted/10 mt-auto justify-end border-t px-6 py-4">
              <Button type="submit">
                <FileCogIcon />
                {t('systemSetting.scheduler.submit')}
              </Button>
            </CardFooter>
          </Card>
        </form>

        <form onSubmit={form.handleSubmit(onSubmit)} className="flex">
          <Card className="w-full gap-4">
            <CardHeader>
              <div className="flex items-center gap-2">
                <EyeOffIcon className="text-primary size-5" />
                <CardTitle>{t('systemSetting.username.title')}</CardTitle>
              </div>
              <CardDescription>{t('systemSetting.username.description')}</CardDescription>
            </CardHeader>
            <CardContent>
              <FormField
                control={form.control}
                name="hideUsername"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <div className="flex items-center justify-between gap-4 rounded-lg border p-4">
                        <Label htmlFor="hide-username" className="font-normal">
                          {field.value
                            ? t('systemSetting.username.yes')
                            : t('systemSetting.username.no')}
                        </Label>
                        <Switch
                          id="hide-username"
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </div>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
            <CardFooter className="bg-muted/10 mt-auto justify-end border-t px-6 py-4">
              <Button type="submit">
                <FileCogIcon />
                {t('systemSetting.username.submit')}
              </Button>
            </CardFooter>
          </Card>
        </form>
      </div>
    </Form>
  )
}
