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

import { source } from "@/lib/source";
import {
  DocsPage,
  DocsBody,
  DocsDescription,
  DocsTitle,
} from "fumadocs-ui/page";
import { notFound } from "next/navigation";
import { createRelativeLink } from "fumadocs-ui/mdx";
import {getMDXComponents} from "@/mdx-components";
import {ArrowUpRightIcon, SquarePenIcon} from "lucide-react";
import {getTranslations, setRequestLocale} from "next-intl/server";

const documentActionClassName = "w-fit border flex items-center gap-2 no-underline rounded-md p-2 font-medium text-sm text-fd-secondary-foreground bg-fd-secondary transition-colors hover:text-fd-accent-foreground hover:bg-fd-accent";

export default async function Page(props: {
  params: Promise<{ lang: string; slug?: string[] }>;
}) {
  const params = await props.params;
  setRequestLocale(params.lang);
  const page = source.getPage(params.slug, params.lang);
  if (!page) notFound();

  const t = await getTranslations("Fumadocs");
  const MDXContent = page.data.body;
  const latestDocumentUrl = `https://raids-lab.github.io/crater/${[
    params.lang,
    "docs",
    ...(params.slug ?? []),
  ].map(encodeURIComponent).join("/")}/`;

  return (
    <DocsPage
      tableOfContent={{
        style: 'clerk',
      }}
      toc={page.data.toc}
      full={page.data.full}
    >
      <DocsTitle>{page.data.title}</DocsTitle>
      <DocsDescription>{page.data.description}</DocsDescription>
      <DocsBody>
        <MDXContent
          components={getMDXComponents({
            // this allows you to link to other pages with relative file paths
            a: createRelativeLink(source, page),
          })}
        />
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <a
            href={`https://github.com/raids-lab/crater/blob/main/website/content/docs/${page.file.path}`}
            rel="noreferrer noopener"
            target="_blank"
            className={documentActionClassName}
          >
            <SquarePenIcon className="size-4" />
            {t("editOnGithub")}
          </a>
          <a
            href={latestDocumentUrl}
            rel="noreferrer noopener"
            target="_blank"
            className={documentActionClassName}
          >
            <ArrowUpRightIcon className="size-4" />
            {t("viewLatestDocs")}
          </a>
        </div>
      </DocsBody>
    </DocsPage>
  );
}

import {locales} from "@/i18n/config";

export async function generateStaticParams() {
  const params = source.generateParams();
  return params.flatMap((p) => {
    const lang = (p as { locale?: string; lang?: string }).locale || (p as { locale?: string; lang?: string }).lang;
    if (lang) {
      return [{ ...p, lang }];
    }
    // 如果 Fumadocs generateParams 没有包含语言信息，则为所有支持的语言生成参数
    return locales.map((l) => ({
      ...p,
      lang: l,
    }));
  });
}

export async function generateMetadata(props: {
  params: Promise<{ lang: string; slug?: string[] }>;
}) {
  const params = await props.params;
  const page = source.getPage(params.slug, params.lang);
  if (!page) notFound();

  return {
    title: page.data.title,
    description: page.data.description,
  };
}
