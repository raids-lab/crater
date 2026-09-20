// Run from the repository root after installing frontend dependencies:
// node --test hack/test-kthena-inference-settings.cjs
// These isolated tests mock React Query and the JSX runtime. They exercise
// request/cache callbacks and rendered state, not browser or framework integration.
const assert = require('node:assert/strict')
const fs = require('node:fs')
const { createRequire } = require('node:module')
const path = require('node:path')
const test = require('node:test')
const vm = require('node:vm')

const root = path.resolve(__dirname, '..')
const ts = createRequire(path.join(root, 'frontend/package.json'))('typescript')
const adminDir = path.join(root, 'frontend/src/routes/admin/more')
const hookPath = path.join(adminDir, '-components/use-kthena-inference-settings.ts')
const viewPath = path.join(adminDir, '-components/kthena-inference-settings.tsx')

function compile(file) {
  const result = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
    fileName: file,
    compilerOptions: {
      target: ts.ScriptTarget.ES2022,
      module: ts.ModuleKind.CommonJS,
      jsx: ts.JsxEmit.ReactJSX,
    },
    reportDiagnostics: true,
  })
  const errors = (result.diagnostics || []).filter(
    (diagnostic) => diagnostic.category === ts.DiagnosticCategory.Error
  )
  assert.equal(errors.length, 0, `${file}: ${JSON.stringify(errors)}`)
  return result.outputText
}

function load(file, mocks) {
  const module = { exports: {} }
  vm.runInNewContext(compile(file), {
    module,
    exports: module.exports,
    require(name) {
      assert.ok(Object.hasOwn(mocks, name), `Unexpected dependency: ${name}`)
      return mocks[name]
    },
  }, { filename: file })
  return module.exports
}

function createHook(queryOverrides = {}, mutationPending = false, invalidate) {
  const calls = { mutations: [], events: [], errors: [] }
  let queryOptions
  let mutationOptions
  const query = {
    data: { enabled: false },
    isSuccess: true,
    isPending: false,
    isError: false,
    isFetching: false,
    refetch: () => calls.events.push('refetch'),
    ...queryOverrides,
  }
  const client = {
    cancelQueries: async ({ queryKey }) => calls.events.push(`cancel:${queryKey.join('/')}`),
    setQueryData: (key, data) => {
      calls.events.push(`cache:${key.join('/')}:${data.enabled}`)
      calls.cached = JSON.parse(JSON.stringify(data))
    },
    invalidateQueries: ({ queryKey }) => {
      calls.events.push(`invalidate:${queryKey.join('/')}`)
      return invalidate ? invalidate(queryKey) : Promise.resolve()
    },
  }
  const mocks = {
    '@tanstack/react-query': {
      useQuery: (options) => { queryOptions = options; return query },
      useQueryClient: () => client,
      useMutation: (options) => {
        mutationOptions = options
        return { isPending: mutationPending, mutate: (value) => calls.mutations.push(value) }
      },
    },
    'react-i18next': { useTranslation: () => ({ t: (key) => key }) },
    sonner: { toast: { success: (key) => calls.events.push(`toast:${key}`) } },
    '@/services/api/system-config': {
      apiAdminGetKthenaInferenceStatus: async () => ({ data: { enabled: true } }),
      apiAdminSetKthenaInferenceStatus: async (enabled) => ({ data: String(enabled) }),
    },
    '@/utils/toast': { showErrorToast: (error) => calls.errors.push(error) },
  }
  const state = load(hookPath, mocks).useKthenaInferenceSettings()
  return { state, calls, queryOptions, mutationOptions }
}

function render(overrides = {}) {
  const calls = { retries: 0 }
  const state = {
    enabled: false,
    isLoading: false,
    isError: false,
    isFetching: false,
    isPending: false,
    refetch: () => { calls.retries += 1 },
    onToggle: () => {},
    ...overrides,
  }
  const element = (type, props) => ({ type, props })
  const mocks = {
    'react/jsx-runtime': { jsx: element, jsxs: element, Fragment: 'Fragment' },
    'react-i18next': { useTranslation: () => ({ t: (key) => key }) },
    'lucide-react': Object.fromEntries(
      ['CheckCircle2Icon', 'Loader2Icon', 'RocketIcon', 'UnplugIcon'].map((key) => [key, key])
    ),
    '@/components/ui/button': { Button: 'Button' },
    '@/components/ui/card': Object.fromEntries(
      ['CardContent', 'CardDescription', 'CardHeader', 'CardTitle'].map((key) => [key, key])
    ),
    '@/components/ui/label': { Label: 'Label' },
    '@/components/ui/switch': { Switch: 'Switch' },
    './use-kthena-inference-settings': { useKthenaInferenceSettings: () => state },
  }
  const tree = load(viewPath, mocks).KthenaInferenceSettings()
  function flatten(node) {
    if (Array.isArray(node)) return node.flatMap(flatten)
    return node && typeof node === 'object' ? [node, ...flatten(node.props.children)] : []
  }
  const nodes = flatten(tree)
  const text = JSON.stringify(tree)
  return { calls, nodes, text, switchNode: nodes.find((node) => node.type === 'Switch') }
}

test('changed TypeScript and TSX files parse', () => {
  for (const file of [hookPath, viewPath, path.join(adminDir, 'index.tsx')]) compile(file)
})

test('admin status query unwraps the response and retains the existing cache key', async () => {
  const { queryOptions } = createHook()
  assert.equal(queryOptions.queryKey.join('/'), 'admin/system-config/kthena-inference')
  assert.equal((await queryOptions.queryFn()).enabled, true)
})

for (const [name, query, pending, requested] of [
  ['initial loading', { data: undefined, isSuccess: false, isPending: true }, false, true],
  ['load error with cached data', { isSuccess: false, isError: true }, false, true],
  ['mutation pending', {}, true, true],
  ['unchanged value', {}, false, false],
]) {
  test(`toggle is ignored during ${name}`, () => {
    const { state, calls } = createHook(query, pending)
    state.onToggle(requested)
    assert.deepEqual(calls.mutations, [])
  })
}

for (const enabled of [true, false]) {
  test(`ready toggle submits ${enabled}`, () => {
    const { state, calls } = createHook({ data: { enabled: !enabled } })
    state.onToggle(enabled)
    assert.deepEqual(calls.mutations, [enabled])
  })
}

test('mutation cancels an in-flight admin status read', async () => {
  const { mutationOptions, calls } = createHook()
  await mutationOptions.onMutate()
  assert.deepEqual(calls.events, ['cancel:admin/system-config/kthena-inference'])
})

test('successful save caches a status object and awaits both cache refreshes', async () => {
  const releases = []
  const { mutationOptions, calls } = createHook({}, false,
    () => new Promise((resolve) => releases.push(resolve)))
  const completion = mutationOptions.onSuccess({ data: 'updated' }, true)
  assert.deepEqual(calls.cached, { enabled: true })
  assert.deepEqual(calls.events, [
    'cache:admin/system-config/kthena-inference:true',
    'invalidate:admin/system-config/kthena-inference',
    'invalidate:system-config/kthena-inference',
  ])
  releases.forEach((release) => release())
  await completion
  assert.equal(calls.events.at(-1), 'toast:systemConfig.kthenaInference.enabledSuccess')
})

test('disable success uses the disabled message and API failures use the error handler', async () => {
  const { mutationOptions, calls } = createHook()
  await mutationOptions.onSuccess({ data: 'updated' }, false)
  assert.deepEqual(calls.cached, { enabled: false })
  assert.equal(calls.events.at(-1), 'toast:systemConfig.kthenaInference.disabledSuccess')
  const error = new Error('request failed')
  mutationOptions.onError(error)
  assert.equal(calls.errors[0], error)
})

test('loading is not presented as a disabled feature', () => {
  const view = render({ enabled: undefined, isLoading: true })
  assert.equal(view.switchNode.props.disabled, true)
  assert.ok(view.nodes.some((node) => node.props.role === 'status'))
  assert.ok(!view.text.includes('disabledWarning'))
  assert.ok(!view.text.includes('activeNotice'))
})

test('failed refresh hides stale status and provides a retry action', () => {
  const view = render({ enabled: true, isError: true })
  assert.equal(view.switchNode.props.disabled, true)
  assert.ok(view.nodes.some((node) => node.props.role === 'alert'))
  assert.ok(!view.text.includes('activeNotice'))
  const retry = view.nodes.find((node) => node.type === 'Button')
  assert.equal(retry.props.disabled, false)
  retry.props.onClick()
  assert.equal(view.calls.retries, 1)
  const refreshing = render({ isError: true, isFetching: true })
  assert.equal(refreshing.nodes.find((node) => node.type === 'Button').props.disabled, true)
})

for (const enabled of [true, false]) {
  test(`confirmed ${enabled} state renders with an accessible switch`, () => {
    const view = render({ enabled })
    const label = view.nodes.find((node) => node.type === 'Label')
    assert.equal(view.switchNode.props.checked, enabled)
    assert.equal(view.switchNode.props.disabled, false)
    assert.equal(label.props.htmlFor, view.switchNode.props.id)
    assert.ok(view.nodes.some(
      (node) => node.props.id === view.switchNode.props['aria-describedby']))
    assert.ok(view.text.includes(enabled ? 'activeNotice' : 'disabledWarning'))
  })
}

test('saving disables the switch and marks the content busy', () => {
  const view = render({ enabled: true, isPending: true })
  assert.equal(view.switchNode.props.disabled, true)
  assert.equal(view.nodes.find((node) => node.type === 'CardContent').props['aria-busy'], true)
})
