import { useMutation, useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
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
})

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

  // set jupyter notebook icon as current page icon
  // set title to jupyter base url
  useEffect(() => {
    if (jupyterInfo?.baseURL && jupyterInfo.fullURL) {
      const link = document.querySelector("link[rel='website icon']") as HTMLLinkElement
      if (link) {
        link.href = `${jupyterInfo.fullURL}/static/favicons/favicon.ico`
        link.type = 'image/x-icon'
      }
      document.title = jupyterInfo.baseURL
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
            <iframe
              title="grafana"
              src={`/portal/job/inter/${name}`}
              height={'100%'}
              width={'100%'}
            />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
