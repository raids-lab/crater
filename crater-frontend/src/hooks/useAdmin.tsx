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

import { useLocation } from 'react-router-dom'

/**
 * useIsAdmin is a custom hook that checks if the current path is an admin view.
 * @returns true if the current path is an admin view, false otherwise
 */
const useIsAdmin = () => {
  const location = useLocation()

  const pathParts = location.pathname.split('/').filter(Boolean)
  const isAdminView = pathParts[0] === 'admin'

  return isAdminView
}

export default useIsAdmin
