import { Outlet, createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

export const Route = createFileRoute('/admin/users')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: <UserManagementCrumb />,
    }
  },
})

function RouteComponent() {
  return <Outlet />
}

function UserManagementCrumb() {
  const { t } = useTranslation()
  return <>{t('navigation.userManagement')}</>
}
