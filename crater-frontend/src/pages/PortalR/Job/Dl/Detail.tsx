import { Card, CardContent } from "@/components/ui/card";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { apiDlTaskInfo } from "@/services/api/dlTask";
import { globalBreadCrumb } from "@/utils/store";
import { useQuery } from "@tanstack/react-query";
import { useEffect, type FC } from "react";
import { useParams } from "react-router-dom";
import { useSetRecoilState } from "recoil";

// route format: /portal/job/ai/detail?id=xxx
const DlJobDetail: FC = () => {
  const setBreadcrumb = useSetRecoilState(globalBreadCrumb);
  const { name: taskName } = useParams();

  const { data: taskInfo, isLoading } = useQuery({
    queryKey: ["dltask", "task", taskName],
    queryFn: () => apiDlTaskInfo(taskName ?? ""),
    select: (res) => res.data.data,
    enabled: !!taskName,
  });

  useEffect(() => {
    if (isLoading) {
      return;
    }
    setBreadcrumb([
      {
        title: "深度推荐训练任务",
        path: "/recommend/job/dl",
      },
      {
        title: `任务「${taskInfo?.name}」详情`,
      },
    ]);
  }, [setBreadcrumb, taskInfo, isLoading]);

  if (isLoading) {
    return <></>;
  }

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

export default DlJobDetail;
