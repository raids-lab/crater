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
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'

import { CopyableCommand } from '@/components/codeblock/copyable-command'
import LogDialog from '@/components/codeblock/log-dialog'
import { NamespacedName } from '@/components/codeblock/pod-container-dialog'
import JupyterIcon from '@/components/icon/jupyter-icon'
import BasicIframe from '@/components/layout/embed/basic-iframe'

import { apiJupyterSnapshot } from '@/services/api/vcjob'
import { queryJupyterToken } from '@/services/query/job'

import FloatingBall from './-components/floating-ball'

export const Route = createFileRoute('/ingress/jupyter/$name')({
  loader: async ({ context: { queryClient }, params: { name } }) => {
    const { data } = await queryClient.ensureQueryData(queryJupyterToken(name))
    if (!data.token || data.token === '') {
      throw new Error(
        'Jupyter token is not available. Please check the job status or try again later.'
      )
    }

    // try to check if connection to Jupyter Notebook is ready
    const url = `${data.fullURL}/api/status`
    if (import.meta.env.DEV) {
      return
    }
    try {
      await ky.get(url, { timeout: 5000 })
    } catch (error) {
      if (error instanceof HTTPError) {
        throw new Error('Jupyter Notebook is not ready yet. Please try again later.')
      }
      throw error
    }
  },
  component: Jupyter,
  errorComponent: Refresh,
})

function Refresh() {
  const { name } = Route.useParams()
  const navigate = useNavigate()
  const [countdown, setCountdown] = useState(10)
  const { data } = useQuery(queryJupyterToken(name ?? ''))

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
            <JupyterIcon className="size-8" />
          </div>
          <CardTitle className="text-xl">Jupyter 连接中...</CardTitle>
          <CardDescription className="text-balance">
            页面将在 <span className="font-mono font-medium text-orange-600">{countdown}</span>{' '}
            秒后自动刷新
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {data && data.urlWithToken && (
            <CopyableCommand label={'Jupyter Lab'} isLink command={data.urlWithToken} />
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

function Jupyter() {
  const { t } = useTranslation()
  // get param from url
  const { name } = Route.useParams()
  const navigate = useNavigate()
  const [namespacedName, setNamespacedName] = useState<NamespacedName>()
  const [isSnapshotOpen, setIsSnapshotOpen] = useState(false)
  const [isDetailOpen, setIsDetailOpen] = useState(false)

  const { data: jupyterInfo } = useSuspenseQuery(queryJupyterToken(name ?? ''))

  // Convert JupyterIcon SVG to favicon and set as page icon
  // set title to jupyter base url
  useEffect(() => {
    if (!jupyterInfo) return
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
    document.title = `${name} - Crater Jupyter`

    // Cleanup function to revoke object URL
    return () => {
      URL.revokeObjectURL(svgUrl)
    }
  }, [jupyterInfo, name])

  const { mutate: snapshot } = useMutation({
    mutationFn: (jobName: string) => apiJupyterSnapshot(jobName),
    onSuccess: () => {
      toast.success(t('jupyter.snapshot.success'))
      navigate({ to: '/portal/env/registry' })
    },
  })

  // drag the floating ball to show log dialog
  const [isDragging, setIsDragging] = useState(false)

  if (!jupyterInfo) {
    throw new Error('Jupyter info is not available')
  }

  return (
    <div className="relative h-screen w-screen">
      <BasicIframe
        title="jupyter notebook"
        key={jupyterInfo.urlWithToken}
        src={jupyterInfo.urlWithToken}
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
            <BasicIframe src={`/portal/jobs/detail/${name}`} height={'100%'} width={'100%'} />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
