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

import "../global.css";
import { RootProvider } from "fumadocs-ui/provider";
import type { ReactNode } from "react";
import {notFound} from 'next/navigation';
import {NextIntlClientProvider, hasLocale, } from 'next-intl';
import OramaSearchDialog from "@/components/search/search";
import { InternalLinkUpdater } from '@/components/internal-link-updater';
import {routing} from '@/i18n/routing';
import {setRequestLocale, getMessages} from 'next-intl/server';
import { locales, localeNames } from '@/i18n/config';

export default async function Layout({
  params,
  children,
}: {
  params: Promise<{ lang: string }>;
  children: ReactNode;
}) {
  const {lang} = await params;

  if (!hasLocale(routing.locales, lang)) {
    notFound();
  }

  setRequestLocale(lang);
  const messages = await getMessages();
  const fumadocsLocales = locales.map((l) => ({
    name: localeNames[l],
    locale: l,
  }));
  
  return (
    <html lang={lang} suppressHydrationWarning>
      <body className="flex flex-col min-h-screen">
        <NextIntlClientProvider locale={lang} messages={messages}>
          <RootProvider
            search={{SearchDialog: OramaSearchDialog}}  
            // 还需要提供语言切换器所需的信息
            i18n={{
              locale: lang,
              locales: fumadocsLocales,
              translations: messages.Fumadocs
            }}

          >
            {children}
            <InternalLinkUpdater />
          </RootProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
