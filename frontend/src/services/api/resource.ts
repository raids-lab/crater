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

import instance, { VERSION } from '../axios'
import { IResponse } from '../types'

export interface Resource {
  ID: number
  name: string
  vendorDomain: string
  resourceType: string
  amount: number
  amountSingleMax: number
  format: string
  priority: number
  label: string
  type?: 'gpu' | 'rdma'
  networks?: Resource[]
}

export const apiResourceList = (withVendorDomain: boolean) => {
  return instance.get<IResponse<Resource[]>>(`${VERSION}/resources`, {
    params: {
      withVendorDomain,
    },
  })
}

// @Router /v1/resources/{id}/networks [get]
export const apiResourceNetworks = (id: number) => {
  return instance.get<IResponse<Resource[]>>(`${VERSION}/resources/${id}/networks`)
}

// @Router /v1/admin/resources/sync [post]
export const apiAdminResourceSync = () => {
  return instance.post<IResponse<never>>(`${VERSION}/admin/resources/sync`)
}

// @Router /v1/admin/resources/{id} [put]
export const apiAdminResourceUpdate = (
  id: number,
  label?: string,
  type?: 'gpu' | 'rdma' | 'default' | null
) => {
  return instance.put<IResponse<Resource>>(`${VERSION}/admin/resources/${id}`, {
    label,
    type,
  })
}

// @Router /v1/admin/operations/patch/pod/resource [put]
export const apiAdminResourceReset = (
  namespace: string | undefined,
  podName: string | undefined,
  key: 'cpu' | 'memory',
  value: string
) => {
  if (key === 'cpu') {
    return instance.put<IResponse<string>>(
      `${VERSION}/admin/namespaces/${namespace}/pods/${podName}/resources`,
      {
        resources: {
          cpu: value,
        },
      }
    )
  } else {
    return instance.put<IResponse<string>>(
      `${VERSION}/admin/namespaces/${namespace}/pods/${podName}/resources`,
      {
        resources: {
          memory: value,
        },
      }
    )
  }
}

// @Router /v1/admin/resources/{id} [delete]
export const apiAdminResourceDelete = (id: number) => {
  return instance.delete<IResponse<Resource>>(`${VERSION}/admin/resources/${id}`)
}

// @Router /v1/admin/resources/{id}/networks [get]
export const apiAdminResourceNetworksList = (id: number) => {
  return instance.get<IResponse<Resource[]>>(`${VERSION}/admin/resources/${id}/networks`)
}

// @Router /v1/admin/resources/{id}/networks [post]
export const apiAdminResourceNetworkAdd = (id: number, rdmaId: number) => {
  return instance.post<IResponse<Resource>>(`${VERSION}/admin/resources/${id}/networks`, {
    rdmaId,
  })
}

// @Router /v1/admin/resources/{id}/networks/{networkId} [delete]
export const apiAdminResourceNetworkDelete = (id: number, networkId: number) => {
  return instance.delete<IResponse<Resource>>(
    `${VERSION}/admin/resources/${id}/networks/${networkId}`
  )
}
