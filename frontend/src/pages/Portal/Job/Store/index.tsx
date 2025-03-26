import DataList from "../../Data/DataList";
import { listJobTemplate } from "@/services/api/jobtemplate";
import { useQuery } from "@tanstack/react-query";
// const mockData: DataItem[] = [
//   {
//     id: 1,
//     name: "作业模板一",
//     desc: "这是第一个作业模板的描述。",
//     username: "用户A",
//     createdAt: "2023-10-01T12:00:00Z",
//     tag: ["数学", "初级"],
//     url: "http://example.com/template1",
//     owner: {
//       id: 1,
//       name: "教师A",
//       nickname: "教师A",
//       email: "teacherA@example.com",
//     },
//   },
//   {
//     id: 2,
//     name: "作业模板二",
//     desc: "这是第二个作业模板的描述。",
//     username: "用户B",
//     createdAt: "2023-10-02T12:00:00Z",
//     tag: ["科学", "中级"],
//     url: "http://example.com/template2",
//     owner: {
//       id: 2,
//       name: "教师B",
//       nickname: "教师B",
//       email: "teacherB@example.com",
//     },
//   },
//   {
//     id: 3,
//     name: "作业模板三",
//     desc: "这是第三个作业模板的描述。",
//     username: "用户C",
//     createdAt: "2023-10-03T12:00:00Z",
//     tag: ["英语", "高级"],
//     url: "http://example.com/template3",
//     owner: {
//       id: 3,
//       name: "教师C",
//       nickname: "教师C",
//       email: "teacherC@example.com",
//     },
//   },
// ];

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
          createdAt: JobTemplate.createdAt,
          owner: JobTemplate.IUserAttributes, // Adjust to match IUserAttributes structure
          username: JobTemplate.IUserAttributes?.nickname || "Unknown User", // Map username
          tag: [], // Map tags if available
        })) || []
      }
      title="作业模板"
    />
  );
}
