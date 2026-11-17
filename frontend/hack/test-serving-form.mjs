import { execFileSync } from 'node:child_process'
import { mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { build } from 'vite'

const directory = await mkdtemp(resolve(tmpdir(), 'crater-serving-test-'))
try {
  const result = await build({
    configFile: false,
    resolve: { alias: { '@': resolve('src') } },
    logLevel: 'error',
    build: {
      write: false,
      minify: false,
      lib: { entry: resolve('hack/serving-form.test.ts'), formats: ['cjs'] },
      rollupOptions: { external: ['node:assert/strict', 'node:test'] },
    },
  })
  const bundle = Array.isArray(result) ? result[0] : result
  const output = resolve(directory, 'test.cjs')
  await writeFile(output, bundle.output[0].code)
  execFileSync(process.execPath, [output], { stdio: 'inherit' })
} finally {
  await rm(directory, { recursive: true, force: true })
}
