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
import { useAtom } from 'jotai'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

import { atomBackendVersion } from '@/utils/store'

/**
 * VersionPage - Version information business component
 *
 * This is a business logic component specific to the Crater platform
 * Displays application version information
 */
export default function VersionPage() {
  const { t } = useTranslation()
  const [backendVersion] = useAtom(atomBackendVersion)

  // Get frontend environment variables
  const frontendAppVersion = import.meta.env.VITE_APP_VERSION
  const frontendCommitSha = import.meta.env.VITE_APP_COMMIT_SHA
  const frontendBuildType = import.meta.env.VITE_APP_BUILD_TYPE
  const frontendBuildTime = import.meta.env.VITE_APP_BUILD_TIME

  return (
    <div className="space-y-6">
      {/* Page title - consistent with other pages */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t('about.title')}</h1>
        <p className="text-muted-foreground">{t('about.description')}</p>
      </div>

      {/* Main content area - four sections with independent spacing */}
      <div className="flex min-h-[70vh] flex-col items-center justify-start pt-4 md:justify-center md:pt-0">
        {/* First section: Logo */}
        <div className="mb-8 flex justify-center md:mb-16">
          <img
            src="/crater.svg"
            alt="Crater"
            className="h-32 w-32 drop-shadow-[0_0_20px_rgba(251,146,60,0.3)] filter md:h-40 md:w-40 dark:drop-shadow-[0_0_25px_rgba(96,165,250,0.4)]"
          />
        </div>

        {/* Second section: Application name and description */}
        <div className="mb-12 space-y-2 text-center md:mb-20 md:space-y-3">
          <h2 className="bg-gradient-to-r from-blue-600 to-cyan-500 bg-clip-text text-5xl font-bold text-transparent md:text-6xl">
            {t('about.appName')}
          </h2>
          <p className="text-muted-foreground text-lg font-semibold md:text-2xl">
            {t('about.appDescription')}
          </p>
        </div>

        {/* Third section: Version information */}
        <div className="mb-12 flex flex-col items-center justify-center gap-4 text-base md:mb-16 md:text-lg">
          {/* Frontend version row */}
          <div className="flex flex-col items-center gap-2 md:flex-row md:gap-8">
            {/* Frontend version */}
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground">{t('about.frontendVersion')}</span>
              <span className="text-foreground font-mono font-semibold">
                {frontendAppVersion || t('about.unavailable')}
              </span>
              {frontendBuildType && frontendBuildType !== 'release' && (
                <Badge variant="secondary" className="text-xs">
                  {t('about.developmentVersion')}
                </Badge>
              )}
            </div>

            {/* Build time */}
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground">{t('about.buildTime')}</span>
              <span className="text-foreground font-mono font-semibold">
                {frontendBuildTime
                  ? new Date(frontendBuildTime).toLocaleString(undefined, {
                      year: 'numeric',
                      month: '2-digit',
                      day: '2-digit',
                      hour: '2-digit',
                      minute: '2-digit',
                    })
                  : t('about.unavailable')}
              </span>
            </div>

            {/* Commit information with GitHub link */}
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground">{t('about.commit')}</span>
              <span className="text-foreground font-mono font-semibold">
                {frontendCommitSha ? frontendCommitSha.substring(0, 7) : t('about.unavailable')}
              </span>
              {frontendCommitSha ? (
                <a
                  href={`https://github.com/raids-lab/crater-frontend/commit/${frontendCommitSha}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-muted-foreground hover:text-foreground transition-colors"
                >
                  <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                  </svg>
                </a>
              ) : (
                <div className="text-muted-foreground/50 cursor-not-allowed">
                  <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                  </svg>
                </div>
              )}
            </div>
          </div>

          {/* Backend version row */}
          <div className="flex flex-col items-center gap-2 md:flex-row md:gap-8">
            {/* Backend version */}
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground">{t('about.backendVersion')}</span>
              <span className="text-foreground font-mono font-semibold">
                {backendVersion?.appVersion || t('about.unavailable')}
              </span>
              {backendVersion?.buildType && backendVersion.buildType !== 'release' && (
                <Badge variant="secondary" className="text-xs">
                  {t('about.developmentVersion')}
                </Badge>
              )}
            </div>

            {/* Backend build time */}
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground">{t('about.buildTime')}</span>
              <span className="text-foreground font-mono font-semibold">
                {backendVersion?.buildTime
                  ? new Date(backendVersion.buildTime).toLocaleString(undefined, {
                      year: 'numeric',
                      month: '2-digit',
                      day: '2-digit',
                      hour: '2-digit',
                      minute: '2-digit',
                    })
                  : t('about.unavailable')}
              </span>
            </div>

            {/* Backend commit information with GitHub link */}
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground">{t('about.commit')}</span>
              <span className="text-foreground font-mono font-semibold">
                {backendVersion?.commitSHA
                  ? backendVersion.commitSHA.substring(0, 7)
                  : t('about.unavailable')}
              </span>
              {backendVersion?.commitSHA ? (
                <a
                  href={`https://github.com/raids-lab/crater-backend/commit/${backendVersion.commitSHA}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-muted-foreground hover:text-foreground transition-colors"
                >
                  <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                  </svg>
                </a>
              ) : (
                <div className="text-muted-foreground/50 cursor-not-allowed">
                  <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                  </svg>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Fourth section: Copyright information */}
        <div className="space-y-2 text-center">
          <div className="text-muted-foreground flex items-center justify-center gap-4 text-sm">
            <span>{t('about.copyright')}</span>
            <span>•</span>
            <a
              href="https://github.com/raids-lab"
              target="_blank"
              rel="noopener noreferrer"
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
              </svg>
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}
