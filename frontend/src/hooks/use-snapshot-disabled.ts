import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { NodeStatus } from '@/services/api/cluster'
import { apiJobGetPods } from '@/services/api/vcjob'
import { queryNodes } from '@/services/query/node'

/**
 * 判断作业所在节点是否被禁止调度，用于禁用保存镜像功能。
 * 若作业的任意 Pod 所在节点处于不可调度或已占用状态，则返回 true。
 */
export function useSnapshotDisabled(name: string | undefined): boolean {
  const { data: pods } = useQuery({
    queryKey: ['job', 'pods', name],
    queryFn: () => apiJobGetPods(name!),
    select: (res) => res.data,
    enabled: !!name,
  })
  const { data: nodes } = useQuery(queryNodes())

  return useMemo(() => {
    if (!pods || !nodes) return false
    const nodeNames = pods.map((p) => p.nodename)
    return nodes.some(
      (n) =>
        nodeNames.includes(n.name) &&
        (n.status === NodeStatus.Unschedulable || n.status === NodeStatus.Occupied)
    )
  }, [pods, nodes])
}
