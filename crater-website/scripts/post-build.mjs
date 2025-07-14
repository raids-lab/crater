import { updateSearchIndexes } from './update-index.mjs';

if (!process.env.CI) {
    const dotenv = await import('dotenv');
    dotenv.config({ path: '.env.local' });
    console.log('Loaded environment variables from .env.local');
}

async function main() {
    console.log('Running post-build script...');
    await Promise.all([updateSearchIndexes()]);
}

await main().catch((e) => {
    console.error('Failed to run post build script', e);
    process.exit(1);
});