import { Outlet, createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

export const Route = createFileRoute('/admin/gpu-analysis')({
  component: RouteComponent,
  // 设置面包屑导航名称，请确保在 i18n 文件中添加了 'navigation.gpuAnalysis'
  loader: () => ({ crumb: t('navigation.gpuAnalysis') }),
})

function RouteComponent() {
  return <Outlet />
}
