import { useMutation, useQuery, useSuspenseQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import ky, { HTTPError } from 'ky'
import { ExternalLink, RefreshCw } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'

import { CopyableCommand } from '@/components/codeblock/copyable-command'
import LogDialog from '@/components/codeblock/log-dialog'
import { NamespacedName } from '@/components/codeblock/pod-container-dialog'
import CodeServerIcon from '@/components/icon/code-server-icon'
import BasicIframe from '@/components/layout/embed/basic-iframe'

import { apiJobSnapshot } from '@/services/api/vcjob'
import { queryWebIDEToken } from '@/services/query/job'

import FloatingBall from './-components/floating-ball'

export const Route = createFileRoute('/ingress/webide/$name')({
  loader: async ({ context: { queryClient }, params: { name } }) => {
    const { data } = await queryClient.ensureQueryData(queryWebIDEToken(name))
    if (!data.token || data.token === '') {
      throw new Error(
        'Jupyter token is not available. Please check the job status or try again later.'
      )
    }

    // try to check if connection to WebIDE Notebook is ready
    const url = `${data.fullURL}/api/status?token=${data.token}`
    if (import.meta.env.DEV) {
      return
    }
    try {
      await ky.get(url, { timeout: 5000 })
    } catch (error) {
      if (error instanceof HTTPError) {
        throw new Error('WebIDE Notebook is not ready yet. Please try again later.')
      }
      throw error
    }
  },
  component: WebIDE,
  errorComponent: Refresh,
})

function Refresh() {
  const { name } = Route.useParams()
  const navigate = useNavigate()
  const [countdown, setCountdown] = useState(10)
  const { data } = useQuery(queryWebIDEToken(name ?? ''))

  // 自动重试倒计时
  useEffect(() => {
    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          window.location.reload()
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [])

  const handleRefresh = () => {
    window.location.reload()
  }

  const handleGoBack = () => {
    navigate({ to: '/portal/jobs/detail/$name', params: { name } })
  }

  return (
    <div className="from-primary to-highlight-violet via-highlight-emerald flex min-h-screen items-center justify-center bg-gradient-to-br">
      <Card className="mx-4 w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex size-8 items-center justify-center rounded-full">
            <CodeServerIcon className="text-primary size-8" />
          </div>
          <CardTitle className="text-xl">WebIDE 连接中...</CardTitle>
          <CardDescription className="text-balance">
            页面将在 <span className="font-mono font-medium text-orange-600">{countdown}</span>{' '}
            秒后自动刷新
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {data && data.fullURL && (
            <CopyableCommand label={'WebIDE Lab'} isLink command={data.fullURL} />
          )}

          <div className="grid grid-cols-2 gap-2">
            <Button onClick={handleRefresh}>
              <RefreshCw className="h-4 w-4 animate-spin" />
              立即刷新页面
            </Button>
            <Button onClick={handleGoBack} variant="outline">
              <ExternalLink className="h-4 w-4" />
              返回任务详情
            </Button>
          </div>

          <div className="text-center text-xs text-gray-500">
            如果长时间无法访问，请检查任务状态或联系管理员
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function WebIDE() {
  const { t } = useTranslation()
  // get param from url
  const { name } = Route.useParams()
  const navigate = useNavigate()
  const [namespacedName, setNamespacedName] = useState<NamespacedName>()
  const [isSnapshotOpen, setIsSnapshotOpen] = useState(false)
  const [isDetailOpen, setIsDetailOpen] = useState(false)
  const [isTokenOpen, setIsTokenOpen] = useState(true)

  const { data: webideInfo } = useSuspenseQuery(queryWebIDEToken(name ?? ''))

  // Convert JupyterIcon SVG to favicon and set as page icon
  // set title to jupyter base url
  useEffect(() => {
    if (!webideInfo) return
    // Create SVG string from JupyterIcon
    const svgString = `
        <svg xmlns="http://www.w3.org/2000/svg" width="192" height="192" viewBox="0 0 192 192" fill="none">
          <style>
            .adaptive-logo {
              fill: #090B0B;
            }

            @media (prefers-color-scheme: dark) {
              .adaptive-logo {
                fill: #ffffff;
              }
            }
          </style>
          <path class="adaptive-logo" d="M55.2598 48C87.2518 48.0001 105.187 64.0911 105.793 87.7764L78.1631 88.6768C77.4359 75.5471 66.469 66.9225 55.2598 67.1797C39.8696 67.5015 28.4776 78.3789 28.4775 95.6279C28.4775 112.877 39.8695 123.562 55.2598 123.562C66.469 123.561 77.1933 115.323 78.4053 102.193L106.035 102.837C105.308 126.908 86.2823 143.256 55.2598 143.256C24.237 143.256 0 124.591 0 95.6279C0.00011362 66.5363 23.2676 48 55.2598 48ZM191.996 50.8145V140.922H119.287V50.8145H191.996Z" />
        </svg>
      `

    // Convert SVG to data URL
    const svgBlob = new Blob([svgString], { type: 'image/svg+xml' })
    const svgUrl = URL.createObjectURL(svgBlob)

    // Update favicon
    let link = document.querySelector("link[rel='icon']") as HTMLLinkElement
    if (!link) {
      link = document.querySelector("link[rel='website icon']") as HTMLLinkElement
    }
    if (!link) {
      link = document.createElement('link')
      link.rel = 'icon'
      document.head.appendChild(link)
    }

    link.href = svgUrl
    link.type = 'image/svg+xml'

    // Set page title
    document.title = `${name} - Crater Jupyter`

    // Cleanup function to revoke object URL
    return () => {
      URL.revokeObjectURL(svgUrl)
    }
  }, [webideInfo, name])

  const { mutate: snapshot } = useMutation({
    mutationFn: (jobName: string) => apiJobSnapshot(jobName),
    onSuccess: () => {
      toast.success(t('jupyter.snapshot.success'))
      navigate({ to: '/portal/env/registry' })
    },
  })

  // drag the floating ball to show log dialog
  const [isDragging, setIsDragging] = useState(false)

  if (!webideInfo) {
    throw new Error('Jupyter info is not available')
  }

  return (
    <div className="relative h-screen w-screen">
      <BasicIframe
        title="jupyter notebook"
        key={webideInfo.fullURL}
        src={webideInfo.fullURL}
        className="absolute top-0 right-0 bottom-0 left-0 h-screen w-screen"
      />
      {/* Transparent overlay */}
      {isDragging && <div className="fixed inset-0 z-50" style={{ cursor: 'move' }} />}
      <FloatingBall
        setIsDragging={setIsDragging}
        handleShowLog={() =>
          webideInfo &&
          setNamespacedName({
            name: webideInfo.podName,
            namespace: webideInfo.namespace,
          })
        }
        handleShowDetail={() => setIsDetailOpen(true)}
        handleSnapshot={() => setIsSnapshotOpen(true)}
        handleShowToken={() => setIsTokenOpen(true)}
      />
      <LogDialog namespacedName={namespacedName} setNamespacedName={setNamespacedName} />
      <AlertDialog open={isSnapshotOpen} onOpenChange={setIsSnapshotOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('jupyter.snapshot.title')}</AlertDialogTitle>
            <AlertDialogDescription>{t('jupyter.snapshot.description')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('jupyter.snapshot.cancel')}</AlertDialogCancel>
            <AlertDialogAction onClick={() => snapshot(name ?? '')}>
              {t('jupyter.snapshot.save')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <Dialog open={isTokenOpen} onOpenChange={setIsTokenOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('floatingBall.tooltip.viewAccessToken')}</DialogTitle>
            <DialogDescription>
              Code Server
              需要输入密码才能登录，如果您不希望使用系统随机生成的密码，可以在作业启动时，通过环境变量传入
              <span className="mx-0.5 font-mono">PASSWORD</span> 选项。
            </DialogDescription>
            <CopyableCommand command={webideInfo.token} />
          </DialogHeader>
        </DialogContent>
      </Dialog>
      <Sheet open={isDetailOpen} onOpenChange={setIsDetailOpen}>
        <SheetContent className="sm:max-w-6xl">
          <SheetHeader>
            <SheetTitle>{t('jupyter.detail.title')}</SheetTitle>
          </SheetHeader>
          <div className="h-[calc(100vh-6rem)] w-full px-4">
            <BasicIframe src={`/portal/jobs/detail/${name}`} height={'100%'} width={'100%'} />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
