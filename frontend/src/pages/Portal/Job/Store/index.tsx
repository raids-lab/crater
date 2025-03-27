import DataList from "../../Data/DataList";
import { listJobTemplate } from "@/services/api/jobtemplate";
import { useQuery } from "@tanstack/react-query";
//import { Link } from "react-router-dom";

export default function AssignmentTemplateList() {
  const data = useQuery({
    queryKey: ["data", "jobtemplate"],
    queryFn: () => listJobTemplate(),
    select: (res) => res.data.data,
  });

  return (
    <DataList
      items={
        data.data?.map((JobTemplate) => ({
          id: JobTemplate.id,
          name: JobTemplate.name,
          desc: JobTemplate.describe,
          tag: [], // Map tags if available
          createdAt: JobTemplate.createdAt,
          template: JobTemplate.template,
          owner: JobTemplate.attribute, // Adjust to match IUserAttributes structure
          username: JobTemplate.attribute?.nickname || "Unknown User", // Map username
        })) || []
      }
      title="作业模板"
      //   actionArea={
      //     {data.data?.map((jobInfo) => (
      //       <Link
      //     ></Link>
      //   }
    />
  );
}
