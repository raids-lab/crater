import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

import { useInferenceKeys } from '@/hooks/inference/use-inference-keys'

import { ServingForm } from './-components/serving-form'

export const Route = createFileRoute('/portal/inference-services/new')({
  validateSearch: (search: Record<string, unknown>) => ({
    clone: typeof search.clone === 'string' ? search.clone : undefined,
    edit: typeof search.edit === 'string' ? search.edit : undefined,
  }),
  component: NewKthenaServicePage,
  loader: () => ({ crumb: t('kthena.form.title') }),
})
function NewKthenaServicePage() {
  const { scope } = useInferenceKeys()
  return <ServingForm key={scope.join(':')} />
}
