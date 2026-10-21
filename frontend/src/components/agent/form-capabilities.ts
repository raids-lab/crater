import type { AgentFormCapability } from '@/components/agent/agent-context'

import type { AgentIdentity } from '@/services/api/agent'

export const agentFormCapabilities = {
  jupyterJob: capability('job.jupyter', '/portal/jobs/new/jupyter-job', 'Jupyter Lab'),
  webIDEJob: capability('job.webide', '/portal/jobs/new/webide-job', 'Web IDE'),
  singleJob: capability('job.single', '/portal/jobs/new/single-job', 'Single Job'),
  pytorchJob: capability('job.pytorch-ddp', '/portal/jobs/new/pytorch-ddp-job', 'PyTorch DDP'),
  tensorflowJob: capability(
    'job.tensorflow-ps',
    '/portal/jobs/new/tensorflow-ps-job',
    'TensorFlow PS'
  ),
  imageEnvd: capability('image.envd', '/portal/env/registry', 'envd Image'),
  imageEnvdRaw: capability('image.envd-raw', '/portal/env/registry', 'envd Script Image'),
  imagePipApt: capability('image.pip-apt', '/portal/env/registry', 'Pip/Apt Image'),
  imageDockerfile: capability('image.dockerfile', '/portal/env/registry', 'Dockerfile Image'),
  imageImport: capability('image.import', '/portal/env/images', 'Image Import'),
  datasetCreate: capability('data.dataset-create', '/portal/data/datasets', 'Dataset'),
  modelCreate: capability('data.model-create', '/portal/data/models', 'Model'),
  modelDownload: capability('data.model-download', '/portal/data/models', 'Model Download'),
  datasetDownload: capability('data.dataset-download', '/portal/data/datasets', 'Dataset Download'),
} as const

export const agentUserNavigationDestinations = [
  '/portal/overview',
  '/portal/monitor/gpu',
  '/portal/monitor/idle',
  '/portal/monitor/network',
  '/portal/jobs/new/jupyter-job',
  '/portal/jobs/new/webide-job',
  '/portal/jobs/new/single-job',
  '/portal/jobs/new/pytorch-ddp-job',
  '/portal/jobs/new/tensorflow-ps-job',
  '/portal/jobs/inter',
  '/portal/jobs/custom',
  '/portal/templates',
  '/portal/env/images',
  '/portal/env/registry',
  '/portal/data/datasets',
  '/portal/data/models',
  '/portal/data/blocks',
  '/portal/files',
  '/portal/more/orders',
  '/portal/more/user',
] as const

export const agentAdminNavigationDestinations = [
  '/admin/jobs',
  '/admin/cluster/nodes',
  '/admin/cluster/resources',
  '/admin/monitor/gpu',
  '/admin/monitor/idle',
  '/admin/monitor/network',
  '/admin/env/registry',
  '/admin/data',
  '/admin/agent',
  '/admin/more',
] as const

export function isAgentNavigationDestination(identity: AgentIdentity, path: string): boolean {
  const destinations: readonly string[] =
    identity === 'admin' ? agentAdminNavigationDestinations : agentUserNavigationDestinations
  return destinations.includes(path)
}

function capability(id: string, route: string, title: string): AgentFormCapability {
  return { id, route, title, schemaVersion: 1 }
}
