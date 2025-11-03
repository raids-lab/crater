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
import { linkOptions, useNavigate } from '@tanstack/react-router'
import { t } from 'i18next'
import { PlusCircleIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import ListedButton, { SplitButtonItem } from '@/components/button/listed-button'

import { JobType } from '@/services/api/vcjob'

export const linkJupyterJob = linkOptions({
  to: '/portal/jobs/new/jupyter-job',
  search: { fromJob: '', fromTemplate: 0 },
  label: t('jobs.new.jupyterJob'),
})

export const linkPyTorchDDPJob = linkOptions({
  to: '/portal/jobs/new/pytorch-ddp-job',
  search: { fromJob: '', fromTemplate: 0 },
  label: t('jobs.new.pytorchDDPJob'),
})

export const linkTensorFlowPSJob = linkOptions({
  to: '/portal/jobs/new/tensorflow-ps-job',
  search: { fromJob: '', fromTemplate: 0 },
  label: t('jobs.new.tensorflowPSJob'),
})

export const linkSingleJob = linkOptions({
  to: '/portal/jobs/new/single-job',
  search: { fromJob: '', fromTemplate: 0 },
  label: t('jobs.new.singleJob'),
})

export const interOptions = [linkJupyterJob]

export const customOptions = [linkPyTorchDDPJob, linkTensorFlowPSJob, linkSingleJob]

export const allOptions = [...interOptions, ...customOptions]

export const getNewJobLink = (jobType: JobType) => {
  switch (jobType) {
    case JobType.Jupyter:
      return linkJupyterJob
    case JobType.Custom:
      return linkSingleJob
    case JobType.Tensorflow:
      return linkTensorFlowPSJob
    case JobType.Pytorch:
      return linkPyTorchDDPJob
    default:
      return linkSingleJob
  }
}

const ListedNewJobButton = ({ mode }: { mode: 'inter' | 'custom' | 'all' }) => {
  const navigate = useNavigate()
  const { t } = useTranslation()

  const getOptionsByMode = (mode: 'inter' | 'custom' | 'all') => {
    switch (mode) {
      case 'inter':
        return interOptions
      case 'custom':
        return customOptions
      case 'all':
        return allOptions
      default:
        return allOptions
    }
  }

  const options = getOptionsByMode(mode)

  const items: SplitButtonItem[] = options.map((option) => ({
    key: option.to,
    title: option.label,
    action: () => navigate(option),
    disabled: false,
  }))

  return (
    <ListedButton
      icon={<PlusCircleIcon className="size-4" />}
      renderTitle={(title) => `${t('jobs.new.action')}${title}`}
      itemTitle="作业类型"
      items={items}
      cacheKey={mode}
    />
  )
}

export default ListedNewJobButton
