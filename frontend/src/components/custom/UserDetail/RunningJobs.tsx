import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const jobs = [
  { id: 1, name: "数据预处理", status: "运行中", progress: "75%" },
  { id: 2, name: "模型训练", status: "等待中", progress: "0%" },
  { id: 3, name: "结果分析", status: "排队中", progress: "0%" },
];

export default function RunningJobs() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>运行中的作业</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>作业名称</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>进度</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {jobs.map((job) => (
              <TableRow key={job.id}>
                <TableCell>{job.name}</TableCell>
                <TableCell>{job.status}</TableCell>
                <TableCell>{job.progress}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
