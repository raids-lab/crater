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
import type { Translations } from "fumadocs-ui/i18n";
import OramaSearchDialog from "@/components/search/search";
import { InternalLinkUpdater } from '@/components/internal-link-updater';

const cn: Partial<Translations> = {
  search: "搜索", // Update: Orama search should be available
  searchNoResult: "没有找到结果",
  toc: "目录",
  tocNoHeadings: "没有标题",
  lastUpdate: "最后更新于",
  chooseLanguage: "选择语言",
  nextPage: "下一页",
  previousPage: "上一页",
  chooseTheme: "选择主题",
  editOnGithub: "在 GitHub 上编辑",
  // other translations
};

const locales = [
  {
    name: "English",
    locale: "en",
  },
  {
    name: "简体中文",
    locale: "cn",
  },
];

export default async function Layout({
  params,
  children,
}: {
  params: Promise<{ lang: string }>;
  children: ReactNode;
}) {
  const lang = (await params).lang;
  return (
    <html lang={lang} suppressHydrationWarning>
      <body className="flex flex-col min-h-screen">
        <RootProvider
          i18n={{
            locale: lang,
            // available languages
            locales,
            // translations for UI
            translations: { cn }[lang],
          }}
          // Orama搜索
          search={{SearchDialog: OramaSearchDialog}}
          
      >
        {children}
        <InternalLinkUpdater />
      </RootProvider>
      </body>
    </html>
  );
}
