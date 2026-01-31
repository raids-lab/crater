'use client';

import { useEffect, useState } from 'react';
import type { SortedResult } from 'fumadocs-core/server';

// ... 你的接口定义保持不变 ...
interface PagefindSearchResult {
  results: {
    id: string;
    data: () => Promise<PagefindResultData>;
  }[];
}

interface PagefindResultData {
  url: string;
  excerpt: string;
  meta: {
    title: string;
  };
  sub_results?: {
    title: string;
    url: string;
    excerpt: string;
  }[];
}

// [!code ++] 新增选项接口
interface PagefindOptions {
  lang?: string;
}

let lastLang: string | undefined;

// [!code warning] 修改函数签名，增加 options 参数
export function usePagefindSearch(search: string, options?: PagefindOptions) {
  const [query, setQuery] = useState<{
    isLoading: boolean;
    data: SortedResult[] | 'empty' | undefined;
  }>({ isLoading: false, data: undefined });

  useEffect(() => {
    // 如果没有搜索词，清空
    if (!search) {
      setQuery({ isLoading: false, data: undefined });
      return;
    }

    const timer = setTimeout(async () => {
      setQuery((prev) => ({ ...prev, isLoading: true }));

      try {
        const basePath = '/crater'; 
        const pagefind = await import(/* webpackIgnore: true */ `${basePath}/pagefind/pagefind.js`);

        if (lastLang && lastLang !== options?.lang) {
          console.log('Language changed from', lastLang, 'to', options?.lang, ', reinitializing Pagefind.');

          await pagefind.destroy();
        }

        await pagefind.options({
          ranking: { termFrequency: 0.5, pageLength: 0.75 },
          excerptLength: 30,
        });

        await pagefind.init();
        lastLang = options?.lang;

        const searchResult: PagefindSearchResult = await pagefind.search(search);
        // [!code highlight:end]

        if (searchResult.results.length === 0) {
          setQuery({ isLoading: false, data: 'empty' });
          return;
        }
        console.log('Pagefind raw results:', searchResult);

        const pages = await Promise.all(
          searchResult.results.slice(0, 5).map((r) => r.data())
        );

        const formatted: SortedResult[] = [];
        
        pages.forEach((page, pageIndex) => {
          let pageUrl = page.url.replace(basePath, '');
          if (!pageUrl.startsWith('/')) pageUrl = '/' + pageUrl;

          formatted.push({
            id: `pg-${pageIndex}`,
            url: pageUrl,
            type: 'page',
            content: page.meta.title || 'Untitled',
          });

          if (!page.sub_results || page.sub_results.length === 0) {
            formatted.push({
              id: `link-${pageIndex}`,
              url: pageUrl,
              type: 'text',
              content: page.excerpt,
            });
            return;
          }

          let lastSectionTitle = '';
          
          page.sub_results.forEach((sub, subIndex) => {
            let subUrl = sub.url.replace(basePath, '');
            if (!subUrl.startsWith('/')) subUrl = '/' + subUrl;

            if (sub.title !== lastSectionTitle && sub.title !== page.meta.title) {
               lastSectionTitle = sub.title;
               formatted.push({
                 id: `head-${pageIndex}-${subIndex}`,
                 url: subUrl, 
                 type: 'heading',
                 content: sub.title,
               });
            }

            formatted.push({
              id: `sub-${pageIndex}-${subIndex}`,
              url: subUrl,
              type: 'text',
              content: sub.excerpt,
            });
          });
        });

        setQuery({ isLoading: false, data: formatted });
      } catch (e) {
        console.error('Pagefind error:', e);
        setQuery({ isLoading: false, data: 'empty' });
      }
    }, 200);

    return () => clearTimeout(timer);
  // [!code warning] 别忘了把 options.lang 加入依赖数组，这样切换语言时会重新搜索
  }, [search, options?.lang]);

  return query;
}