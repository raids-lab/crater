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
import { useQuery } from '@tanstack/react-query'
import { GithubIcon, StarIcon, XIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useLocalStorage } from 'usehooks-ts'

import { Button } from '@/components/ui/button'

const GITHUB_REPO = 'raids-lab/crater'
const GITHUB_URL = `https://github.com/${GITHUB_REPO}`
const DISMISSED_KEY = 'github-star-card-dismissed'

export function GitHubStarCard() {
  const { t, i18n } = useTranslation()
  const [isDismissed, setIsDismissed] = useLocalStorage(DISMISSED_KEY, false)

  const { data: starCount } = useQuery({
    queryKey: ['github-stars', GITHUB_REPO],
    enabled: !isDismissed,
    queryFn: async ({ signal }) => {
      try {
        const response = await fetch(`https://api.github.com/repos/${GITHUB_REPO}`, { signal })
        if (!response.ok) return null
        const data = await response.json()
        return typeof data.stargazers_count === 'number' &&
          Number.isInteger(data.stargazers_count) &&
          data.stargazers_count >= 0
          ? data.stargazers_count
          : null
      } catch {
        return null
      }
    },
    staleTime: 1000 * 60 * 60, // 1 hour
    gcTime: 1000 * 60 * 60 * 24, // 24 hours
  })

  if (isDismissed) return null

  return (
    <div className="mx-auto mb-6 grid w-full max-w-3xl grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-3 rounded-lg border border-orange-500/20 bg-orange-500/10 px-5 py-3.5 sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:px-6">
      <div className="flex min-w-0 items-start gap-3">
        <StarIcon
          className="mt-0.5 size-5 shrink-0 fill-amber-500 text-amber-500"
          aria-hidden="true"
        />
        <p className="text-sm leading-relaxed">{t('about.star.description')}</p>
      </div>
      <Button
        variant="outline"
        size="sm"
        className="col-start-1 row-start-2 justify-self-start sm:col-start-2 sm:row-start-1"
        asChild
      >
        <a href={GITHUB_URL} target="_blank" rel="noopener noreferrer">
          <GithubIcon aria-hidden="true" />
          <span>{t('about.star.action')}</span>
          {starCount !== null && starCount !== undefined && (
            <span
              className="ml-1 border-l pl-2 text-xs tabular-nums"
              aria-label={t('about.star.count', { count: starCount })}
            >
              {starCount.toLocaleString(i18n.resolvedLanguage)}
            </span>
          )}
        </a>
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="text-muted-foreground col-start-2 row-start-1 size-7 self-start sm:col-start-3 sm:self-center"
        onClick={() => setIsDismissed(true)}
        aria-label={t('about.star.dismiss')}
      >
        <XIcon aria-hidden="true" />
      </Button>
    </div>
  )
}
