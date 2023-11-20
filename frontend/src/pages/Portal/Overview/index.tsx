import type { FC } from "react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
    <div className="space-y-1 px-6 py-6 text-xl">
      <p>用户登录到系统后，最先看到的页面。</p>
      <p>集中显示用户的近期项目、资源使用情况、任务进度等。</p>
      <div className="grid grid-flow-row-dense grid-cols-2 gap-4 pt-6 md:grid-cols-4">
        <Card className="col-span-2 h-48">
          <CardHeader>
            <CardTitle>当前任务摘要</CardTitle>
          </CardHeader>
          <CardContent>
            <p>如果不存在，则引导用户新建任务</p>
          </CardContent>
        </Card>
        <Card className="col-span-1 h-48">
          <CardHeader>
            <CardTitle>资源占用</CardTitle>
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
        <Card className="col-span-1 h-48">
          <CardHeader>
            <CardTitle>使用建议</CardTitle>
          </CardHeader>
          <CardContent>
            <p>链接使用文档</p>
          </CardContent>
        </Card>
        <Card className="col-span-full h-80">
          <CardHeader>
            <CardTitle>近期项目</CardTitle>
          </CardHeader>
          <CardContent>
            <p>显示用户近期的项目，点击进入项目详情</p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};
