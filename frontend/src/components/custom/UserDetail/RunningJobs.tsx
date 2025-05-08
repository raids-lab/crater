// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function RunningJobs() {
  const { t } = useTranslation();

  const jobs = [
    {
      id: 1,
      name: t("jobs.dataPreprocessing"),
      status: t("statuses.running"),
      progress: "75%",
    },
    {
      id: 2,
      name: t("jobs.modelTraining"),
      status: t("statuses.waiting"),
      progress: "0%",
    },
    {
      id: 3,
      name: t("jobs.resultAnalysis"),
      status: t("statuses.queued"),
      progress: "0%",
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("runningJobs.title")}</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("runningJobs.headers.jobName")}</TableHead>
              <TableHead>{t("runningJobs.headers.status")}</TableHead>
              <TableHead>{t("runningJobs.headers.progress")}</TableHead>
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
