import { BotIcon, Loader2Icon, RotateCcwIcon, SaveIcon } from 'lucide-react'
import { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

// 引入 AlertDialog 组件
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'

// 导出 Schema 供父组件使用
export const createLlmSettingsSchema = (t: (key: string) => string) =>
  z.object({
    baseUrl: z.string().url({ message: t('systemConfig.llm.validation.url') }),
    modelName: z.string().min(1, { message: t('systemConfig.llm.validation.model') }),
    apiKey: z.string().optional(),
  })

export type LlmFormSchema = z.infer<ReturnType<typeof createLlmSettingsSchema>>
interface LlmSettingsProps {
  form: UseFormReturn<LlmFormSchema>
  isPending: boolean
  onSubmit: (values: LlmFormSchema, validate: boolean) => void
  onReset: () => void
}

export function LlmSettings({ form, isPending, onSubmit, onReset }: LlmSettingsProps) {
  const { t } = useTranslation()

  return (
    <>
      <CardHeader>
        <div className="flex items-center gap-2">
          <BotIcon className="text-primary h-5 w-5" />
          <CardTitle>{t('systemConfig.llm.title')}</CardTitle>
        </div>
        <CardDescription>{t('systemConfig.llm.description')}</CardDescription>
      </CardHeader>
      <Form {...form}>
        <form>
          <CardContent className="space-y-4">
            <FormField
              control={form.control}
              name="baseUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('systemConfig.llm.baseUrl')}</FormLabel>
                  <FormControl>
                    <Input placeholder="https://api.openai.com/v1" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="modelName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('systemConfig.llm.modelName')}</FormLabel>
                  <FormControl>
                    <Input placeholder="gpt-4o / deepseek-chat" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="apiKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('systemConfig.llm.apiKey')}</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="sk-..." {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
          <CardFooter className="bg-muted/10 flex flex-wrap justify-between gap-4 px-6 py-4">
            <p className="text-muted-foreground text-xs italic">{t('systemConfig.llm.tip')}</p>
            <div className="flex gap-2">
              {/* 使用 AlertDialog 包裹重置按钮 */}
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button type="button" variant="destructive" disabled={isPending}>
                    <RotateCcwIcon className="mr-2 h-4 w-4" />
                    {t('common.reset')}
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    {/* 这里使用了假设的翻译key，请根据实际情况调整 */}
                    <AlertDialogTitle>
                      {t('systemConfig.llm.resetConfirmTitle') || '确认重置配置？'}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {t('systemConfig.llm.resetConfirmDesc') ||
                        '此操作将清除当前表单的所有修改并恢复为默认值，无法撤销。'}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                    {/* 点击确认时触发 onReset */}
                    <AlertDialogAction onClick={onReset}>{t('common.confirm')}</AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>

              <Button
                type="button"
                onClick={form.handleSubmit((values) => onSubmit(values, true))}
                disabled={isPending}
              >
                {isPending ? (
                  <Loader2Icon className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <SaveIcon className="mr-2 h-4 w-4" />
                )}
                {t('common.save')}
              </Button>
            </div>
          </CardFooter>
        </form>
      </Form>
      <Separator />
    </>
  )
}
