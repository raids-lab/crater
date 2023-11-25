import type { FC } from "react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useQuery } from "@tanstack/react-query";
import { apiAiTaskQuota } from "@/services/api/aiTask";
import { getAiResource } from "@/utils/resource";

export const Component: FC = () => {
  const { data: quota, isLoading } = useQuery({
    queryKey: ["aitask", "quota"],
    queryFn: apiAiTaskQuota,
    select: (res) => ({
      hard: getAiResource(res.data.data.hard),
      hardUsed: getAiResource(res.data.data.hardUsed),
    }),
  });

  return (
    <div className="grid grid-flow-row-dense grid-cols-2 gap-6 md:grid-cols-2">
      <Card className="col-span-1 h-60">
        <CardHeader>
          <CardTitle>作业队列</CardTitle>
          <CardDescription>
            <p>1. 所有正在系统中运行的作业</p>
            <p>2. (如果有) 用户自己正在运行的作业</p>
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p></p>
        </CardContent>
      </Card>
      <Card className="col-span-1 h-60">
        <CardHeader>
          <CardTitle>查看配额</CardTitle>
          <CardDescription>
            <p>1. 查看用户个人账户的资源配额</p>
            <p>2. 查看用户在不同账户下的资源配额</p>
          </CardDescription>
        </CardHeader>
        <CardContent className="font-mono text-base">
          {isLoading ? (
            <p>loading...</p>
          ) : (
            quota && (
              <>
                <p>
                  CPU: {quota.hardUsed.cpu}/{quota.hard.cpu}
                </p>
                <p>
                  GPU: {quota.hardUsed.gpu}/{quota.hard.gpu}
                </p>
                <p>
                  MEM: {quota.hardUsed.memory}/{quota.hard.memory}
                </p>
              </>
            )
          )}
        </CardContent>
      </Card>
      <Card className="col-span-full h-80">
        <CardHeader>
          <CardTitle>资源使用情况</CardTitle>
          <CardDescription>
            <p>1. 当前整体集群的资源使用情况</p>
            <p>2. 查看整体集群的历史资源使用情况 (详细信息对接prometheus)</p>
            <p>3. 查看本用户的历史资源使用情况 (汇总统计)</p>
          </CardDescription>
        </CardHeader>
        <CardContent></CardContent>
      </Card>
    </div>
  );
};
