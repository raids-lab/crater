import DataList from "../../Data/DataList";
import { listJobTemplate, deleteJobTemplate } from "@/services/api/jobtemplate";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import DocsButton from "@/components/button/DocsButton";
export default function AssignmentTemplateList() {
  // const [loading, setLoading] = useState(true);
  // 获取 queryClient 实例，用于手动刷新数据
  const queryClient = useQueryClient();

  // 使用 React Query 来获取作业模板数据
  const { data: templateData } = useQuery({
    queryKey: ["data", "jobtemplate"],
    queryFn: () => listJobTemplate(),
    select: (res) => res.data.data,
  });

  const { mutate: handleDelete } = useMutation({
    mutationFn: (id: number) => deleteJobTemplate(id),
    onSuccess: () => {
      // 删除成功后，刷新数据
      queryClient.invalidateQueries({ queryKey: ["data", "jobtemplate"] });
    },
  });

  return (
    <DataList
      items={
        templateData?.map((jobTemplate) => ({
          id: jobTemplate.id,
          name: jobTemplate.name,
          desc: jobTemplate.describe,
          tag: [], // 假设 API 返回的是 tags 字段
          createdAt: jobTemplate.createdAt,
          template: jobTemplate.template,
          owner: jobTemplate.userInfo,
        })) || []
      }
      title="作业模板"
      handleDelete={handleDelete}
      actionArea={
        <div className="flex flex-row gap-3">
          <DocsButton title={"作业模板文档"} url="quick-start/jobtemplate" />
        </div>
      }
    />
  );
}
