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

import Image from 'next/image';
import { getChartVersion, isChartVersionInjected } from '@/lib/crater-version';

/**
 * A simple badge showing the Crater Helm Chart version.
 * Used primarily in headers of chart documentation.
 */
export function ChartBadge() {
  const version = getChartVersion();
  const showVersion = isChartVersionInjected();

  return (
    <div className="not-prose flex items-center gap-2 mt-1 mb-6">
      <Image
        src="https://img.shields.io/badge/Type-application-informational?style=flat-square"
        alt="Type: application"
        width={105}
        height={20}
        unoptimized
      />
      {showVersion && (
        <span className="font-mono text-sm font-bold bg-fd-secondary text-fd-secondary-foreground px-2 py-0.5 rounded border">
          v{version}
        </span>
      )}
    </div>
  );
}
