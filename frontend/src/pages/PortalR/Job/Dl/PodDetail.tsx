import { Card, CardContent } from "@/components/ui/card";
import { apiDlTaskPods } from "@/services/api/recommend/dlTask";
import { globalBreadCrumb } from "@/utils/store";
import { useQuery } from "@tanstack/react-query";
import { useEffect, type FC } from "react";
import { useParams } from "react-router-dom";
import { useSetRecoilState } from "recoil";

const PodDetail: FC = () => {
  const setBreadcrumb = useSetRecoilState(globalBreadCrumb);
  const { name: taskName, podName } = useParams();

  const podInfo = useQuery({
    queryKey: ["dltask", "task", taskName, "pod"],
    queryFn: () => apiDlTaskPods(taskName ?? ""),
    select: (res) => res.data.data,
    enabled: !!taskName,
  });

  useEffect(() => {
    if (!taskName) {
      return;
    }
    setBreadcrumb([
      {
        title: "深度推荐训练作业",
        path: "/recommend/job/dl",
      },
      {
        title: `作业「${taskName}」详情`,
        path: `/recommend/job/dl/${taskName}`,
      },
      {
        title: `Pod 详情`,
      },
    ]);
  }, [setBreadcrumb, taskName]);
  return (
    <div className="grid grid-cols-2 gap-4">
      <h2 className="col-span-full text-base font-bold">Pod 详情</h2>
      {podInfo.data && (
        <Card className="col-span-full pt-6">
          <CardContent>
            <pre className="whitespace-pre-wrap text-xs">
              {JSON.stringify(
                podInfo.data.find((item) => item.metadata.name === podName),
                null,
                2,
              )}
            </pre>
          </CardContent>
        </Card>
      )}
    </div>
  );
};

export default PodDetail;
