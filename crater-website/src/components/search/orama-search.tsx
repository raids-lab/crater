// components/search/orama-search.tsx
'use client';

import { OramaSearchBox} from '@orama/react-components';
import { useI18n } from 'fumadocs-ui/contexts/i18n';
import type { SharedProps } from 'fumadocs-ui/components/dialog/search';
import { useEffect, useState, useMemo} from 'react';
import { useTheme } from "next-themes";

// 从环境变量中获取 Orama 配置
const endpoint = process.env.NEXT_PUBLIC_ORAMA_ENDPOINT;
const apiKey = process.env.NEXT_PUBLIC_ORAMA_PUBLIC_API_KEY;

// 搜索配置
const searchBoxConfig = {
    sourcesMap: {
        path: "url",
        title: "section",
        description: "title"
    },
    resultMap: {
        path: "url",
        title: "section",
        description: "content",
        section: "title"
    },
    themeConfig: {},
};

// 翻译
const oramaDictionaryEn = {
    searchPlaceholder: "Search our documentation...",
    chatPlaceholder: "Ask our AI assistant...",
    noResultsFound: "We couldn't find any results",
    noResultsFoundFor: "for",
    suggestionsTitle: "You might want to try",
    seeAll: "View all results",
    addMore: "Load more",
    clearChat: "Reset conversation",
    errorMessage: "Oops! Something went wrong with your search.",
    disclaimer: "AI-generated responses may not always be accurate.",
    startYourSearch: "Begin your search",
    initErrorSearch: "Search service initialization failed",
    initErrorChat: "Chat service initialization failed",
    chatButtonLabel: "Get AI summary"
};

const oramaDictionaryCn = {
    searchPlaceholder: "搜索文档...",
    chatPlaceholder: "询问 AI 助手...",
    noResultsFound: "没有找到结果",
    noResultsFoundFor: "关于",
    suggestionsTitle: "你或许想试试",
    seeAll: "查看所有结果",
    addMore: "加载更多",
    clearChat: "重置对话",
    errorMessage: "哎呀！搜索出错了。",
    disclaimer: "AI 生成的回复可能并不总是准确。",
    startYourSearch: "开始搜索吧",
    initErrorSearch: "搜索服务初始化失败",
    initErrorChat: "聊天服务初始化失败",
    chatButtonLabel: "获取 AI 摘要"
};

const oramaDictionaries: Record<string, typeof oramaDictionaryEn> = { // 使用 typeof 来确保结构一致
    en: oramaDictionaryEn,
    cn: oramaDictionaryCn,
};

//搜索建议
const oramaSuggestions = {
    en: [
        "How to start a PyTorch job?",
        "How to connect via SSH?",
        "How to perform LLM inference on Crater?"
    ],
    cn: [
        "怎么启动Pytorch job？",
        "怎么进行SSH连接？",
        "怎么在Crater上进行LLM推理？"
    ]
};

export default function OramaSearchDialog({ open, onOpenChange }: SharedProps) { // ▲ FIX 1: 解构 onOpenChange
    const { locale } = useI18n();

    const [isClient, setIsClient] = useState(false);
    useEffect(() => {
        setIsClient(true);
    }, []);

    const currentOramaDictionary = useMemo(() => {
        return oramaDictionaries[locale as keyof typeof oramaDictionaries] || oramaDictionaries.en; // 默认为英文
    }, [locale]);

    const currentSuggestions = useMemo(() => {
        return oramaSuggestions[locale as keyof typeof oramaSuggestions] || oramaSuggestions.en;
    }, [locale]);

    const { resolvedTheme } = useTheme();
    const oramaColorScheme = resolvedTheme as 'light' | 'dark' | undefined;

    if (!isClient || !endpoint || !apiKey) {
        if (typeof window !== 'undefined' && (!endpoint || !apiKey)) {
            console.error('Orama environment variables are missing.');
        }
        return null;
    }

    return (
        <OramaSearchBox
            {...searchBoxConfig}

            open={open}
            onModalClosed={() => onOpenChange(false)}

            index={{
                endpoint: endpoint,
                api_key: apiKey,
            }}
            searchParams={{
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                where: {
                    locale: locale,
                }  as never,
            }}
            sourceBaseUrl= {process.env.NEXT_PUBLIC_SOURCE_BASE_URL || ''}
            dictionary={currentOramaDictionary}
            showKeyboardShortcuts={false}
            colorScheme={oramaColorScheme}
            suggestions={currentSuggestions}
            relatedQueries={3}
            clearChatOnDisconnect={true}
            highlightTitle={{}}
            highlightDescription={{}}
        />
    );
}