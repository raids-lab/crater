'use client';

import { useState, useMemo } from 'react';
import {
  SearchDialog,
  SearchDialogContent,
  SearchDialogHeader,
  SearchDialogInput,
  SearchDialogList,
  SearchDialogOverlay,
  SearchDialogFooter,
  type SharedProps,
  SearchDialogListItem
} from 'fumadocs-ui/components/dialog/search';
import { cn } from '@/lib/utils'; // 假设你有这个工具函数，如果没有直接写字符串也可以
import { Hash } from 'lucide-react'; // 可选：引入图标增强视觉
import { usePagefindSearch } from './use-pagefind'; // 你的 hook 路径
import { useLocale, useTranslations } from 'next-intl';

export default function CustomSearchDialog(props: SharedProps) {
  const [search, setSearch] = useState('');
  const locale = useLocale();
  const t = useTranslations('PagefindSearch');

  const query = usePagefindSearch(search, { lang: locale });

  // 模拟原生风格的键盘按键样式
function Shortcut({ children }: { children: React.ReactNode }) {
  return (
    <kbd className="inline-flex h-5 select-none items-center gap-1 rounded border border-fd-border bg-fd-background px-1.5 font-mono text-[10px] font-medium text-fd-muted-foreground opacity-100">
      {children}
    </kbd>
  );
}

// Pagefind 的 Logo (简化的风筝/搜索形状)
function PagefindIcon({ className }: { className?: string }) {
  return (
    <svg 
      viewBox="0 0 106 110" 
      fill="currentColor" 
      className={className}
      aria-hidden="true"
    >
      <path d="M79.46 32.22A42.17 42.17 0 0 0 52.88 5.61a42.27 42.27 0 0 0-38 15.17l-1.42 1.83 23.49 23.49a12.85 12.85 0 0 1 12.44-8 12.63 12.63 0 0 1 9.38 4.14L79.46 32.22Zm-29 33.72a12.79 12.79 0 0 1-10.82 6.09 12.63 12.63 0 0 1-9.36-4.13L9.61 88.58a42.18 42.18 0 0 0 26.58 26.62 42.27 42.27 0 0 0 38-15.18l1.42-1.83L52.1 74.71a12.85 12.85 0 0 1-1.64-8.77ZM27.07 19.34a42.12 42.12 0 0 0-9.82 7.74 42.27 42.27 0 0 0-4.82 40.58l1.83 2.76 23.49-23.49a12.87 12.87 0 0 1-.68-27.59Zm69.31 71.32a42.11 42.11 0 0 0 9.82-7.73 42.27 42.27 0 0 0 4.82-40.58l-1.83-2.76-23.49 23.49a12.87 12.87 0 0 1 .68 27.58Z" />
    </svg>
  );
}

  // --- 关键步骤：转换数据以支持高亮 ---
  const items = useMemo(() => {
    if (!query.data || query.data === 'empty') return [];

    return query.data.map((item) => ({
      ...item,
      // 使用 dangerouslySetInnerHTML 渲染 Pagefind 返回的带 <mark> 的 HTML
      // 同时使用 Tailwind 的任意变体语法 [&>mark] 来美化 mark 标签的样式
      content: (
        <span
          className="[&>mark]:bg-fd-primary/20 [&>mark]:text-fd-primary [&>mark]:font-medium [&>mark]:rounded-sm [&>mark]:px-0.5"
          dangerouslySetInnerHTML={{ __html: item.content }}
        />
      ),
    }));
  }, [query.data]);

  return (
    <SearchDialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      search={search}
      onSearchChange={setSearch}
      isLoading={query.isLoading}
    >
      <SearchDialogOverlay />
      <SearchDialogContent>
        <SearchDialogHeader>
          <SearchDialogInput />
        </SearchDialogHeader>
        
    
    <SearchDialogList 
      items={items}
      // [!code ++] 添加自定义 Item 渲染器
      Item={({ item, onClick }) => {
        return (
          <SearchDialogListItem
            key={item.id}
            item={item}
            onClick={onClick}
            // 根据类型调整样式
            className={cn(
              // 基础样式调整
              "gap-2", 
              
              // === 针对 Heading (锚点 #) 的样式 ===
              // 稍微缩进，加粗
              item.type === 'heading' && "pl-4 font-medium",

              // === 针对 Text (正文内容) 的样式 ===
              // 大幅缩进，颜色变淡，左侧加边框模拟树状结构
              item.type === 'text' && "ml-4 pl-4 border-l-2 border-fd-border text-fd-muted-foreground text-sm"
            )}
          >
            {/* 
              可选：如果你想手动控制图标，可以在这里写。
              如果 items 里面已经处理好了 content (包含高亮HTML)，直接渲染 children 即可
              SearchDialogListItem 默认会渲染 item.content
            */}
            
            {/* 示例：给 Heading 加个小图标 */}
            {item.type === 'heading' && (
              <Hash className="size-3 text-fd-muted-foreground shrink-0 translate-y-[1px]" />
            )}
            
            <div className="flex-1 truncate">
               {item.content}
            </div>
          </SearchDialogListItem>
        );
      }}
    />

        <SearchDialogFooter className="flex w-full items-center justify-between border-t border-fd-border bg-fd-secondary/30 px-3 py-2">
          {/* 左侧：快捷键提示 */}
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1.5 text-xs text-fd-muted-foreground">
              <Shortcut>ESC</Shortcut>
              <span>{t('close')}</span>
            </div>
            <div className="flex items-center gap-1.5 text-xs text-fd-muted-foreground">
              <div className="flex gap-0.5">
                <Shortcut>↑</Shortcut>
                <Shortcut>↓</Shortcut>
              </div>
              <span>{t('select')}</span>
            </div>
          </div>

          {/* 右侧：Pagefind 品牌标识 */}
          <a
            href="https://pagefind.app"
            target="_blank"
            rel="noreferrer noopener"
            className="flex items-center gap-1.5 text-xs text-fd-muted-foreground transition-colors hover:text-fd-foreground"
          >
            <span>{t('poweredBy')}</span>
            <span className="font-semibold text-fd-foreground">Pagefind</span>
            <PagefindIcon className="size-3.5 text-fd-foreground" />
          </a>
        </SearchDialogFooter>
      </SearchDialogContent>
    </SearchDialog>
  );
}