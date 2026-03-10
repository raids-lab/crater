/**
 * Copyright 2025 Crater
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

"use client";

import React, { ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import { getChartVersion, isChartVersionInjected, CHART_YAML_URL } from '@/lib/crater-version';
import { Callout } from 'fumadocs-ui/components/callout';

const chartYamlLink = (chunks: ReactNode) => (
  <a href={CHART_YAML_URL} target="_blank" rel="noopener noreferrer">
    {chunks}
  </a>
);

/**
 * Renders information about the Crater Helm Chart version and its
 * requirement for image version consistency.
 */
export function CraterChartVersionNotice() {
  const t = useTranslations('CraterChartVersionNotice');
  const version = getChartVersion();
  const link = chartYamlLink;

  const content = isChartVersionInjected() ? (
    <>
      {t.rich('message', { link })}
      {' '}
      <span className="font-mono font-bold">
        {t('version', { version })}
      </span>
    </>
  ) : (
    <>{t.rich('fallback', { link })}</>
  );

  return (
    <Callout type="info" className="mt-2 mb-4">
      {content}
    </Callout>
  );
}
