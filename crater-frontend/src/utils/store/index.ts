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

import { IUserAttributes } from '@/services/api/admin/user'
import { IBackendVersionInfo, IUserContext } from '@/services/api/auth'

/**
 * LocalStorage and Jotai Keys
 */
const LAST_VIEW_KEY = 'last_view'
const SETTINGS_KEY = 'settings'
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
