// components/search/orama-search.tsx
'use client';

import { OramaSearchBox} from '@orama/react-components';
import type { SharedProps } from 'fumadocs-ui/components/dialog/search';
import { useEffect, useState } from 'react';
import { useTheme } from "next-themes";
import { useTranslations, useLocale } from 'next-intl';

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

export default function OramaSearchDialog({ open, onOpenChange }: SharedProps) { // ▲ FIX 1: 解构 onOpenChange
    const t = useTranslations('OramaSearch');
    const locale = useLocale();

    const [isClient, setIsClient] = useState(false);
    useEffect(() => {
        setIsClient(true);
    }, []);

    const dictionary = {
        searchPlaceholder: t('dictionary.searchPlaceholder'),
        chatPlaceholder: t('dictionary.chatPlaceholder'),
        noResultsFound: t('dictionary.noResultsFound'),
        noResultsFoundFor: t('dictionary.noResultsFoundFor'),
        suggestionsTitle: t('dictionary.suggestionsTitle'),
        seeAll: t('dictionary.seeAll'),
        addMore: t('dictionary.addMore'),
        clearChat: t('dictionary.clearChat'),
        errorMessage: t('dictionary.errorMessage'),
        disclaimer: t('dictionary.disclaimer'),
        startYourSearch: t('dictionary.startYourSearch'),
        initErrorSearch: t('dictionary.initErrorSearch'),
        initErrorChat: t('dictionary.initErrorChat'),
        chatButtonLabel: t('dictionary.chatButtonLabel'),
    };
    const suggestions = Array.from({ length: 3 }, (_, i) => t(`suggestions.${i}`));

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
            dictionary={dictionary}
            showKeyboardShortcuts={false}
            colorScheme={oramaColorScheme}
            suggestions={suggestions}
            relatedQueries={3}
            clearChatOnDisconnect={true}
            highlightTitle={{}}
            highlightDescription={{}}
        />
    );
}