// scripts/update-index.mjs

import { sync } from 'fumadocs-core/search/orama-cloud';
import * as fs from 'node:fs/promises';
import { CloudManager } from '@oramacloud/client';

// 移除了函数返回类型 ": Promise<void>"
export async function updateSearchIndexes() {
    const apiKey = process.env.ORAMA_PRIVATE_API_KEY;
    const indexName = process.env.ORAMA_INDEX_NAME;

    if (!apiKey || !indexName) {
        console.log('no api key for Orama found, skipping');
        return;
    }

    const content = await fs.readFile('.next/server/app/api/search.body');
    const records = JSON.parse(content.toString());

    const manager = new CloudManager({ api_key: apiKey });

    await sync(manager, {
        index: indexName,
        documents: records,
    });

    console.log(`search updated: ${records.length} records`);
}