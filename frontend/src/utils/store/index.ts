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
import { atom } from 'jotai'
import { atomWithStorage, useResetAtom } from 'jotai/utils'

import type {
  AgentActionContext,
  AgentFormRuntime,
  AgentResourceContext,
} from '@/components/agent/agent-context'

import { IUserAttributes } from '@/services/api/admin/user'
import { IBackendVersionInfo, IUserContext } from '@/services/api/auth'

/**
 * LocalStorage and Jotai Keys
 */
const LAST_VIEW_KEY = 'last_view'
const SETTINGS_KEY = 'settings'
const PRIVACY_ACCEPTED_KEY = 'privacy_accepted' // 新增：隐私政策同意状态
const AGENT_FLOATING_BUTTON_PREFERENCES_KEY = 'agent_floating_button_preferences'

export const ACCESS_TOKEN_KEY = 'access_token'
export const REFRESH_TOKEN_KEY = 'refresh_token'
export const VITE_UI_THEME_KEY = 'vite_ui_theme'

/**
 * Navigation BreadCrumb
 */
export type BreadCrumbItem = {
  href: string
  label: string
  back?: boolean
}

export const atomBreadcrumb = atom([] as BreadCrumbItem[])

export const atomFixedLayout = atom(false)

/**
 * User Context
 */
export type UserInfo = IUserAttributes & {
  space: string
}

export const atomUserInfo = atom<UserInfo>()
export const atomUserContext = atom<IUserContext>()
export const atomBackendVersion = atom<IBackendVersionInfo>()

/**
 * Crater-Agent panel state and per-user browser-local floating entry preferences.
 * Portal and admin identities share the same preference for one user, while
 * different users on the same browser remain isolated.
 */
export const atomAgentPanelOpen = atom(false)
export const atomAgentActionContext = atom<AgentActionContext>()
export const atomAgentFormRuntime = atom<AgentFormRuntime>()
export const atomAgentPageResourceContext = atom<AgentResourceContext>()
const atomAgentFloatingButtonPreferences = atomWithStorage<Record<string, boolean>>(
  AGENT_FLOATING_BUTTON_PREFERENCES_KEY,
  {},
  undefined,
  { getOnInit: true }
)
export const atomAgentUserFloatingButtonEnabled = atom(
  (get) => {
    const userID = get(atomUserInfo)?.id
    if (!userID) return true
    return get(atomAgentFloatingButtonPreferences)[String(userID)] ?? true
  },
  (get, set, enabled: boolean) => {
    const userID = get(atomUserInfo)?.id
    if (!userID) return
    const preferences = get(atomAgentFloatingButtonPreferences)
    set(atomAgentFloatingButtonPreferences, {
      ...preferences,
      [String(userID)]: enabled,
    })
  }
)

/**
 * Remember the last view.
 * Will not be cleared when logout.
 */
export const globalLastView = atomWithStorage(LAST_VIEW_KEY, '', undefined, {
  getOnInit: true,
})

export const globalSettings = atomWithStorage(
  SETTINGS_KEY,
  {
    scheduler: 'volcano' as 'volcano' | 'colocate' | 'sparse',
    hideUsername: false,
  },
  undefined,
  {
    getOnInit: true,
  }
)

/**
 * 是否已在当前浏览器同意隐私政策
 * true 表示用户已在本设备上确认过，登录页可默认勾选
 */
export const atomPrivacyAccepted = atomWithStorage<boolean>(
  PRIVACY_ACCEPTED_KEY,
  false,
  undefined,
  {
    getOnInit: true,
  }
)

export const globalJobUrl = atom((get) => {
  const scheduler = get(globalSettings).scheduler
  switch (scheduler) {
    case 'volcano':
      return 'vcjobs'
    case 'colocate':
      return 'aijobs'
    case 'sparse':
      return 'spjobs'
    default:
      return 'vcjobs'
  }
})

export const globalHideUsername = atom((get) => {
  const hideUsername = get(globalSettings).hideUsername
  return hideUsername
})

/**
 * Reset all states
 */
export const useResetStore = () => {
  const resetSettings = useResetAtom(globalSettings)

  const resetAll = () => {
    // Jotai
    resetSettings()
  }

  return { resetAll }
}
