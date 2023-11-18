import { Card, CardContent } from "@/components/ui/card";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { apiAiTaskGet, convertAiTask } from "@/services/api/aiTask";
import { globalBreadCrumb } from "@/utils/store";
import { showErrorToast } from "@/utils/toast";
import { useQuery } from "@tanstack/react-query";
import { useEffect, type FC } from "react";
import { useParams } from "react-router-dom";
import { useSetRecoilState } from "recoil";

// route format: /portal/job/ai/detail?id=xxx
const AiJobDetail: FC = () => {
  const setBreadcrumb = useSetRecoilState(globalBreadCrumb);
  const { id: taskID } = useParams();
  const { data: taskInfo, isLoading } = useQuery({
    queryKey: ["aitask", "task", taskID],
    retry: 1,
    queryFn: () => apiAiTaskGet(parseInt(taskID ?? "")),
    select: (res) => {
      const taskInfo = convertAiTask(res.data.data);
      return taskInfo;
    },
    onError: (err) => showErrorToast("获取任务列表失败", err),
  });

  useEffect(() => {
    if (isLoading) return;
    setBreadcrumb([
      {
        title: "AI 训练任务",
        path: "/portal/job/ai",
      },
      {
        title: `任务「${taskInfo?.taskName}」详情`,
      },
    ]);
  }, [setBreadcrumb, isLoading, taskInfo]);

  return (
    <div className="space-y-4 px-6 py-4">
      <Card>
        <ScrollArea className="w-full">
          <CardContent className="w-full max-w-screen-sm pt-6">
            <pre className="text-clip text-sm">
              {JSON.stringify(taskInfo, null, 2)}
            </pre>
          </CardContent>
          <ScrollBar orientation="horizontal" className="hidden" />
        </ScrollArea>
      </Card>
    </div>
  );
};

export default AiJobDetail;
