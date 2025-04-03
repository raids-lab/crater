import DataList from "../../Data/DataList";
import { listJobTemplate, deleteJobTemplate } from "@/services/api/jobtemplate";
import { useQuery, useQueryClient } from "@tanstack/react-query";

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

  // 定义刷新数据的函数
  const refreshData = () => {
    // 使用 queryClient 的 invalidateQueries 方法使查询缓存失效，触发重新获取
    queryClient.invalidateQueries({ queryKey: ["data", "jobtemplate"] });
  };

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
          owner: jobTemplate.attribute, // 调整以匹配 IUserAttributes 结构
          username: jobTemplate.attribute?.nickname || "Unknown User", // 映射用户名
        })) || []
      }
      title="作业模板"
      itemdelete={deleteJobTemplate}
      onRefresh={refreshData} // 传入刷新函数
      //   actionArea={
      //     {data.data?.map((jobInfo) => (
      //       <Link
      //     ></Link>
      //   }
    />
  );
}
