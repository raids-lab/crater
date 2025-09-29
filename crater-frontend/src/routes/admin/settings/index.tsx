/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
// i18n-processed-v1.1.0
import { zodResolver } from '@hookform/resolvers/zod'
import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'
import { useAtom } from 'jotai'
import { FileCogIcon } from 'lucide-react'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import WarningAlert from '@/components/custom/warning-alert'

import { globalSettings } from '@/utils/store'

export const Route = createFileRoute('/admin/settings/')({
  component: RouteComponent,
  loader: () => ({ crumb: t('navigation.platformSettings') }),
})

function RouteComponent() {
  const { t } = useTranslation()

  // Moved Zod schema to component
  const formSchema = z.object({
    scheduler: z.enum(['volcano', 'colocate', 'sparse'], {
      invalid_type_error: t('systemSetting.scheduler.invalidType'),
      required_error: t('systemSetting.scheduler.required'),
    }),
    hideUsername: z.boolean().default(false),
  })

  type FormSchema = z.infer<typeof formSchema>

  const [settings, setSettings] = useAtom(globalSettings)

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: settings,
  })

  const handleSubmit = () => {
    toast.success(t('systemSetting.toast.success'))
    setSettings(form.getValues())
    // refresh page
    window.location.reload()
  }

  return (
    <>
      <WarningAlert
        title={t('systemSetting.warning.title')}
        description={t('systemSetting.warning.description')}
      />
      <Card>
        <CardHeader>
          <CardTitle>{t('systemSetting.scheduler.title')}</CardTitle>
          <CardDescription>{t('systemSetting.scheduler.description')}</CardDescription>
        </CardHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)}>
            <CardContent>
              <FormField
                control={form.control}
                name="scheduler"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Select onValueChange={field.onChange} defaultValue={field.value}>
                        <SelectTrigger className="">
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
            <CardFooter className="px-6 py-4">
              <Button type="submit">
                <FileCogIcon />
                {t('systemSetting.scheduler.submit')}
              </Button>
            </CardFooter>
          </form>
        </Form>
        <CardHeader>
          <CardTitle>{t('systemSetting.username.title')}</CardTitle>
          <CardDescription>{t('systemSetting.username.description')}</CardDescription>
        </CardHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)}>
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
                        <SelectTrigger className="">
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
            <CardFooter className="px-6 py-4">
              <Button type="submit">
                <FileCogIcon />
                {t('systemSetting.username.submit')}
              </Button>
            </CardFooter>
          </form>
        </Form>
      </Card>
    </>
  )
}
