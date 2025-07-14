'use client';

import dynamic from 'next/dynamic';
import type { SharedProps } from 'fumadocs-ui/components/dialog/search';

const OramaSearchDialog = dynamic(
    () => import('./orama-search'),
    {
        ssr: false,
        loading: () => <div className="p-4">Loading Search...</div>
    }
);

export default function SafeOramaSearchDialog(props: SharedProps) {
    return <OramaSearchDialog {...props} />;
}