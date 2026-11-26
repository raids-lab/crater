import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { compileFunction } from 'node:vm'
import ts from 'typescript'

function loadModule(path, dependencies) {
  const source = readFileSync(new URL(path, import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS },
  })
  const exports = {}
  compileFunction(outputText, ['exports', 'require'], { filename: path })(exports, (name) => {
    assert.ok(Object.hasOwn(dependencies, name), `Unexpected dependency: ${name}`)
    return dependencies[name]
  })
  return exports
}

const cluster = loadModule('../src/services/api/cluster.ts', { '@/services/client': {} })
const { NodeStatus } = cluster
const {
  deriveNodeStatus,
  matchesNodeStatusFilter,
  NODE_TAINT_ACCOUNT,
  NODE_TAINT_NOT_READY,
  NODE_TAINT_UNREACHABLE,
  NODE_TAINT_UNSCHEDULABLE,
} = loadModule('../src/components/node/node-status-utils.ts', { '@/services/api/cluster': cluster })

const accountTaint = {
  key: NODE_TAINT_ACCOUNT,
  value: 'research',
  effect: 'NoSchedule',
  timeAdded: '2026-11-25T12:00:00Z',
}
const notReadyTaint = { key: NODE_TAINT_NOT_READY, effect: 'NoExecute' }
const unreachableTaint = { key: NODE_TAINT_UNREACHABLE, effect: 'NoExecute' }
const cordonTaint = { key: NODE_TAINT_UNSCHEDULABLE, effect: 'NoSchedule' }

test('an occupied node matches NotReady before the not-ready taint is added', () => {
  const node = { status: NodeStatus.Occupied, readyStatus: 'False', taints: [accountTaint] }

  assert.deepEqual(deriveNodeStatus(node).tags, [NodeStatus.NotReady, NodeStatus.Occupied])
  assert.equal(matchesNodeStatusFilter(node, [NodeStatus.NotReady]), true)
  assert.equal(matchesNodeStatusFilter(node, [NodeStatus.Ready]), false)
})

test('an occupied node matches Unschedulable before the cordon taint is added', () => {
  const node = { status: NodeStatus.Occupied, unschedulable: true, taints: [accountTaint] }

  assert.deepEqual(deriveNodeStatus(node).tags, [
    NodeStatus.Ready,
    NodeStatus.Unschedulable,
    NodeStatus.Occupied,
  ])
  assert.equal(matchesNodeStatusFilter(node, [NodeStatus.Unschedulable]), true)
})

test('legacy cordon detection requires the exact taint key and NoSchedule effect', () => {
  const node = {
    status: NodeStatus.Ready,
    taints: [
      { key: 'ops.node.kubernetes.io/unschedulable', effect: 'PreferNoSchedule' },
      { key: 'ops.node.kubernetes.io/unschedulable', effect: 'NoSchedule' },
      { key: NODE_TAINT_UNSCHEDULABLE, effect: 'PreferNoSchedule' },
    ],
  }

  assert.equal(deriveNodeStatus(node).unschedulable, null)
  assert.equal(matchesNodeStatusFilter(node, [NodeStatus.Unschedulable]), false)
})

test('an explicit schedulable value takes precedence over the taint and legacy status', () => {
  const node = { status: NodeStatus.Unschedulable, unschedulable: false, taints: [cordonTaint] }

  assert.equal(deriveNodeStatus(node).unschedulable, null)
  assert.equal(matchesNodeStatusFilter(node, [NodeStatus.Unschedulable]), false)
})

test('readiness takes precedence and retains only the matching taint for details', () => {
  const node = { status: NodeStatus.Unknown, taints: [notReadyTaint, unreachableTaint] }

  assert.deepEqual(deriveNodeStatus({ ...node, readyStatus: 'True' }).health, {
    state: NodeStatus.Ready,
  })
  assert.deepEqual(deriveNodeStatus({ ...node, readyStatus: 'False' }).health, {
    state: NodeStatus.NotReady,
    taint: notReadyTaint,
  })
  assert.deepEqual(deriveNodeStatus({ ...node, readyStatus: 'Unknown' }).health, {
    state: NodeStatus.Unknown,
    taint: unreachableTaint,
  })
})

test('pressure tags merge API conditions, legacy status and exact taints once per condition', () => {
  const memoryTaint = { key: 'node.kubernetes.io/memory-pressure', effect: 'NoSchedule' }
  const pidTaint = { key: 'node.kubernetes.io/pid-pressure', effect: 'NoSchedule' }
  const node = {
    status: NodeStatus.NetworkUnavailable,
    readyStatus: 'True',
    pressures: [NodeStatus.DiskPressure, NodeStatus.MemoryPressure],
    taints: [memoryTaint, pidTaint, { key: 'ops.node.kubernetes.io/disk-pressure' }],
  }

  assert.deepEqual(deriveNodeStatus(node).pressures, [
    { state: NodeStatus.MemoryPressure, taint: memoryTaint },
    { state: NodeStatus.DiskPressure, taint: undefined },
    { state: NodeStatus.PIDPressure, taint: pidTaint },
    { state: NodeStatus.NetworkUnavailable, taint: undefined },
  ])
  assert.equal(matchesNodeStatusFilter(node, [NodeStatus.NotReady, NodeStatus.DiskPressure]), true)
})

test('legacy taints and annotations still supply health, scheduling and one occupied account', () => {
  const node = {
    status: NodeStatus.Occupied,
    taints: [
      { ...accountTaint, effect: 'PreferNoSchedule', value: 'other' },
      accountTaint,
      unreachableTaint,
      cordonTaint,
    ],
    annotations: {
      'crater.raids.io/taint-reason-occupied': ' planned allocation ',
      'crater.raids.io/taint-operator-occupied': 'admin',
    },
  }
  const derived = deriveNodeStatus(node)

  assert.deepEqual(derived.tags, [
    NodeStatus.Unknown,
    NodeStatus.Unschedulable,
    NodeStatus.Occupied,
  ])
  assert.deepEqual(derived.occupied, {
    account: 'research',
    reason: 'planned allocation',
    operator: 'admin',
    timeAdded: accountTaint.timeAdded,
  })
})

test('legacy status still supplies badges when the corresponding taints are missing', () => {
  for (const status of [
    NodeStatus.NotReady,
    NodeStatus.Unknown,
    NodeStatus.MemoryPressure,
    NodeStatus.Unschedulable,
    NodeStatus.Occupied,
  ]) {
    const node = { status, taints: [] }
    assert.equal(matchesNodeStatusFilter(node, [status]), true, status)
  }
})
