/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { QueryClient, QueryObserver } from '@tanstack/react-query'
import assert from 'node:assert/strict'
import test from 'node:test'

import { shouldReadJobAccessPods } from './pod-read-policy.ts'

const runningSingleJob = {
  jobName: 'ordinary-job',
  isEMIASJob: false,
  isRunning: true,
  isSingleJob: true,
}

test('running ordinary single jobs keep their access Pod read enabled', () => {
  assert.equal(shouldReadJobAccessPods(runningSingleJob), true)
})

test('hidden access controls never enable legacy Pod reads', () => {
  for (const isEMIASJob of [false, true]) {
    for (const isRunning of [false, true]) {
      for (const isSingleJob of [false, true]) {
        assert.equal(
          shouldReadJobAccessPods({ ...runningSingleJob, isEMIASJob, isRunning, isSingleJob }),
          !isEMIASJob && isRunning && isSingleJob
        )
      }
    }
  }
  assert.equal(shouldReadJobAccessPods({ ...runningSingleJob, jobName: '' }), false)
})

test('EMIAS and distributed job queries do not issue requests on mount', () => {
  const client = new QueryClient()
  let calls = 0
  const fetchPods = async () => {
    calls++
    return []
  }
  try {
    for (const job of [
      { ...runningSingleJob, isEMIASJob: true },
      { ...runningSingleJob, isSingleJob: false },
      { ...runningSingleJob, isRunning: false },
    ]) {
      const observer = new QueryObserver(client, {
        queryKey: ['access-pods', job],
        queryFn: fetchPods,
        enabled: shouldReadJobAccessPods(job),
      })
      const unsubscribe = observer.subscribe(() => {})
      assert.equal(observer.getCurrentResult().fetchStatus, 'idle')
      unsubscribe()
    }
    assert.equal(calls, 0)
  } finally {
    client.clear()
  }
})

test('access Pod reads start when a single job runs and stop after it finishes', async () => {
  const client = new QueryClient()
  let calls = 0
  const fetchPods = async () => {
    calls++
    return [{ namespace: 'test', name: 'ordinary-job-0' }]
  }
  const options = (isRunning: boolean) => ({
    queryKey: ['access-pods', runningSingleJob.jobName],
    queryFn: fetchPods,
    enabled: shouldReadJobAccessPods({ ...runningSingleJob, isRunning }),
  })
  const observer = new QueryObserver(client, options(false))
  const unsubscribe = observer.subscribe(() => {})
  try {
    assert.equal(calls, 0)
    const loaded = new Promise<void>((resolve) => {
      const stop = observer.subscribe((result) => {
        if (result.isSuccess) {
          stop()
          resolve()
        }
      })
    })
    observer.setOptions(options(true))
    await loaded
    assert.equal(calls, 1)
    assert.deepEqual(observer.getCurrentResult().data, [
      { namespace: 'test', name: 'ordinary-job-0' },
    ])
    observer.setOptions(options(false))
    await client.invalidateQueries({ queryKey: options(false).queryKey })
    assert.equal(calls, 1)
  } finally {
    unsubscribe()
    client.clear()
  }
})
