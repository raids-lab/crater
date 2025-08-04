import { useMutation, useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { ExternalLink, RefreshCw } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
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
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'

import LogDialog from '@/components/codeblock/LogDialog'
import { NamespacedName } from '@/components/codeblock/PodContainerDialog'

import { apiJupyterSnapshot } from '@/services/api/vcjob'
import { queryJupyterToken } from '@/services/query/job'

import FloatingBall from './-components/FloatingBall'

export const Route = createFileRoute('/ingress/jupyter/$name')({
  loader: ({ context: { queryClient }, params: { name } }) => {
    return queryClient.ensureQueryData(queryJupyterToken(name))
  },
  component: Jupyter,
  errorComponent: Refresh,
})

function Refresh() {
  const { name } = Route.useParams()
  const navigate = useNavigate()
  const [countdown, setCountdown] = useState(10)

  // 自动重试倒计时
  useEffect(() => {
    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          window.location.reload()
          return 5
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
    <div className="from-highlight-orange to-highlight-fuchsia via-highlight-red flex min-h-screen items-center justify-center bg-gradient-to-br">
      <Card className="mx-4 w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-orange-100">
            <RefreshCw className="h-6 w-6 animate-spin text-orange-600" />
          </div>
          <CardTitle className="text-xl">Jupyter 服务正在启动</CardTitle>
          <CardDescription className="text-balance">
            Jupyter Notebook 服务可能还未完全就绪，请稍等片刻或手动刷新页面
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="text-center text-sm text-gray-600">
            页面将在 <span className="font-mono font-medium text-orange-600">{countdown}</span>{' '}
            秒后自动刷新
          </div>

          <div className="flex flex-col space-y-2">
            <Button onClick={handleRefresh} className="w-full">
              <RefreshCw className="mr-2 h-4 w-4" />
              立即刷新
            </Button>
            <Button onClick={handleGoBack} variant="outline" className="w-full">
              <ExternalLink className="mr-2 h-4 w-4" />
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

function Jupyter() {
  const { t } = useTranslation()
  // get param from url
  const { name } = Route.useParams()
  const navigate = useNavigate()
  const [namespacedName, setNamespacedName] = useState<NamespacedName>()
  const [isSnapshotOpen, setIsSnapshotOpen] = useState(false)
  const [isDetailOpen, setIsDetailOpen] = useState(false)

  const { data: jupyterInfo } = useQuery(queryJupyterToken(name))

  const url = useMemo(() => {
    if (jupyterInfo) {
      return `${jupyterInfo.fullURL}?token=${jupyterInfo.token}`
    }
    return ''
  }, [jupyterInfo])

  // Convert JupyterIcon SVG to favicon and set as page icon
  // set title to jupyter base url
  useEffect(() => {
    if (jupyterInfo?.baseURL) {
      // Create SVG string from JupyterIcon
      const svgString = `
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32">
          <path
            fill="#f57c00"
            d="M6.2 18a22.7 22.7 0 0 0 9.8 2 22.7 22.7 0 0 0 9.8-2 10.002 10.002 0 0 1-19.6 0m19.6-4a22.7 22.7 0 0 0-9.8-2 22.7 22.7 0 0 0-9.8 2 10.002 10.002 0 0 1 19.6 0"
          />
          <circle cx="27" cy="5" r="3" fill="#757575" />
          <circle cx="5" cy="27" r="3" fill="#9e9e9e" />
          <circle cx="5" cy="5" r="3" fill="#616161" />
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
      document.title = `${jupyterInfo.baseURL} - Crater Jupyter`

      // Cleanup function to revoke object URL
      return () => {
        URL.revokeObjectURL(svgUrl)
      }
    }
  }, [jupyterInfo])

  const { mutate: snapshot } = useMutation({
    mutationFn: (jobName: string) => apiJupyterSnapshot(jobName),
    onSuccess: () => {
      toast.success(t('jupyter.snapshot.success'))
      navigate({ to: '/portal/env/registry' })
    },
  })

  // drag the floating ball to show log dialog
  const [isDragging, setIsDragging] = useState(false)

  return (
    <div className="relative h-screen w-screen">
      <iframe
        title="jupyter notebook"
        src={url}
        className="absolute top-0 right-0 bottom-0 left-0 h-screen w-screen"
      />
      {/* Transparent overlay */}
      {isDragging && <div className="fixed inset-0 z-50" style={{ cursor: 'move' }} />}
      <FloatingBall
        setIsDragging={setIsDragging}
        handleShowLog={() =>
          jupyterInfo &&
          setNamespacedName({
            name: jupyterInfo.podName,
            namespace: jupyterInfo.namespace,
          })
        }
        handleShowDetail={() => setIsDetailOpen(true)}
        handleSnapshot={() => setIsSnapshotOpen(true)}
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
      <Sheet open={isDetailOpen} onOpenChange={setIsDetailOpen}>
        <SheetContent className="sm:max-w-6xl">
          <SheetHeader>
            <SheetTitle>{t('jupyter.detail.title')}</SheetTitle>
          </SheetHeader>
          <div className="h-[calc(100vh-6rem)] w-full px-4">
            <iframe src={`/portal/jobs/detail/${name}`} height={'100%'} width={'100%'} />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
