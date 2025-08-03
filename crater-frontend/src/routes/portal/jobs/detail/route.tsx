import { Outlet, createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/portal/jobs/detail')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: 'Job Detail',
      nolink: true, // Prevents the breadcrumb from being a link
    }
  },
})

function RouteComponent() {
  return <Outlet />
}
