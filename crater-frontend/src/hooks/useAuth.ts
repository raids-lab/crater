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

import { Role } from '@/services/api/auth'
import { globalAccount } from '@/utils/store'
import { useMemo } from 'react'
import { useAtomValue } from 'jotai'

export interface UserInfo {
  name: string // unique username
  space: string // user space
}

/**
 * A hook that checks if the current user is authenticated and has the required role.
 *
 * @param requireRole - The required role to access the resource.
 * @returns A boolean indicating whether the user is authenticated and has the required role.
 */
export function useAuth(requireRole: Role) {
  const { rolePlatform: role } = useAtomValue(globalAccount)
  const isAuthenticated = useMemo(() => {
    switch (requireRole) {
      case Role.Admin:
        return role === Role.Admin
      case Role.User:
        return role === Role.Admin || role === Role.User
      case Role.Guest:
        return true
      default:
        return true
    }
  }, [requireRole, role])
  return isAuthenticated
}
