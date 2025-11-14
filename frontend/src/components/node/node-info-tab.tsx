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
import {
  Cable,
  CpuIcon,
  Grid,
  HardDriveIcon,
  Layers,
  MemoryStickIcon as Memory,
  ServerIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'

import { IClusterNodeDetail, IClusterNodeGPU } from '@/services/api/cluster'

type NodeInfoTabProps = {
  nodeDetail?: IClusterNodeDetail
  gpuDetail?: IClusterNodeGPU
}

export const NodeInfoTab = ({ nodeDetail, gpuDetail }: NodeInfoTabProps) => {
  const { t } = useTranslation()

  if (!nodeDetail) return null

  // 解析资源信息
  const cpuCount = nodeDetail.capacity?.cpu || '-'
  const memoryCapacity = nodeDetail.capacity?.memory
    ? `${(parseFloat(nodeDetail.capacity.memory) / (1024 * 1024 * 1024)).toFixed(2)} GB`
    : '-'
  const diskCapacity = nodeDetail.capacity?.['ephemeral-storage']
    ? `${(parseFloat(nodeDetail.capacity['ephemeral-storage']) / (1024 * 1024 * 1024)).toFixed(2)} GB`
    : '-'

  return (
    <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
      {/* 基础信息卡片 */}
      <Card>
        <CardContent className="space-y-3 px-6 py-2">
          <h3 className="text-lg font-semibold">{t('nodeDetail.info.basicInfo')}</h3>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground flex items-center gap-2">
                <Memory className="size-4" />
                <span>{t('nodeDetail.info.totalMemory')}</span>
              </div>
              <Badge variant="outline" className="font-mono text-xs">
                {memoryCapacity}
              </Badge>
            </div>
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground flex items-center gap-2">
                <HardDriveIcon className="size-4" />
                <span>{t('nodeDetail.info.totalDisk')}</span>
              </div>
              <Badge variant="outline" className="font-mono text-xs">
                {diskCapacity}
              </Badge>
            </div>
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground flex items-center gap-2">
                <ServerIcon className="size-4" />
                <span>{t('nodeDetail.info.kernelVersion')}</span>
              </div>
              <Badge variant="outline" className="font-mono text-xs">
                {nodeDetail.kernelVersion || '-'}
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* CPU 信息卡片 */}
      <Card>
        <CardContent className="space-y-3 px-6 py-2">
          <h3 className="text-lg font-semibold">{t('nodeDetail.info.cpuInfo')}</h3>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground flex items-center gap-2">
                <Grid className="size-4" />
                <span>{t('nodeDetail.info.architecture')}</span>
              </div>
              <Badge variant="outline" className="font-mono text-xs">
                {nodeDetail.arch.toUpperCase()}
              </Badge>
            </div>
            <div className="flex items-center justify-between">
              <div className="text-muted-foreground flex items-center gap-2">
                <CpuIcon className="size-4" />
                <span>{t('nodeDetail.info.cpuCount')}</span>
              </div>
              <Badge variant="outline" className="font-mono text-xs">
                {cpuCount}
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 加速卡信息卡片 - 仅在有 GPU 时显示 */}
      {gpuDetail?.haveGPU && (
        <Card className="md:col-span-2">
          <CardContent className="space-y-3 px-6 py-2">
            <h3 className="text-lg font-semibold">{t('nodeDetail.info.acceleratorInfo')}</h3>
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <div className="flex items-center justify-between">
                <div className="text-muted-foreground flex items-center gap-2">
                  <ServerIcon className="size-4" />
                  <span>{t('nodeDetail.gpu.product')}</span>
                </div>
                <Badge variant="outline" className="font-mono text-xs">
                  {gpuDetail.gpuProduct}
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <div className="text-muted-foreground flex items-center gap-2">
                  <Grid className="size-4" />
                  <span>{t('nodeDetail.gpu.count')}</span>
                </div>
                <Badge variant="outline" className="font-mono text-xs">
                  {gpuDetail.gpuCount}
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <div className="text-muted-foreground flex items-center gap-2">
                  <Memory className="size-4" />
                  <span>{t('nodeDetail.gpu.memory')}</span>
                </div>
                <Badge variant="outline" className="font-mono text-xs">
                  {parseInt(gpuDetail.gpuMemory) / 1024} GB
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <div className="text-muted-foreground flex items-center gap-2">
                  <Layers className="size-4" />
                  <span>{t('nodeDetail.gpu.architecture')}</span>
                </div>
                <Badge variant="outline" className="font-mono text-xs">
                  {gpuDetail.gpuArch}
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <div className="text-muted-foreground flex items-center gap-2">
                  <Cable className="size-4" />
                  <span>{t('nodeDetail.gpu.driverVersion')}</span>
                </div>
                <Badge variant="outline" className="font-mono text-xs">
                  {gpuDetail.gpuDriver}
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <div className="text-muted-foreground flex items-center gap-2">
                  <Cable className="size-4" />
                  <span>CUDA {t('nodeDetail.info.version')}</span>
                </div>
                <Badge variant="outline" className="font-mono text-xs">
                  {gpuDetail.cudaVersion}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

export default NodeInfoTab
