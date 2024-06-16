import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import { RefreshCcwIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import {
  TableHead,
  TableRow,
  TableHeader,
  TableCell,
  TableBody,
  Table,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { IJupyterDetail, JobPhase } from "@/services/api/vcjob";
import { apiJupyterGetAdminDetail } from "@/services/api/vcjob";
export interface Resource {
  [key: string]: string;
}

const JupyterDetail = () => {
  const initial: IJupyterDetail = {
    name: "",
    namespace: "",
    username: "",
    jobName: "",
    retry: "",
    queue: "",
    status: JobPhase.Pending,
    createdAt: "",
    startedAt: "",
    runtime: "",
    podDetails: [],
    useTensorBoard: false,
  };
  const { id } = useParams<string>();
  const jobName = "" + id;
  const setBreadcrumb = useBreadcrumb();
  const [data, setData] = useState<IJupyterDetail>(initial);
  const [refetchInterval, setRefetchInterval] = useState(5000); // Manage interval state
  const queryClient = useQueryClient();

  const { data: taskList, isLoading } = useQuery({
    queryKey: ["job", "detail", jobName],
    queryFn: () => apiJupyterGetAdminDetail(jobName),
    select: (res) => res.data.data,
    refetchInterval: refetchInterval,
  });

  const handleIntervalChange = (newInterval: string) => {
    const intervalMs = parseInt(newInterval) * 1000; // Convert seconds to milliseconds
    setRefetchInterval(intervalMs);
  };

  const handleResourceData = (resourceJson: string): Resource => {
    try {
      return JSON.parse(resourceJson) as Resource;
    } catch (error) {
      return {};
    }
  };
  useEffect(() => {
    setBreadcrumb([{ title: "作业详情" }]);
  }, [setBreadcrumb]);

  useEffect(() => {
    if (!isLoading && taskList) {
      setData(taskList);
    }
  }, [taskList, isLoading, taskList?.runtime]);

  interface PodDetail {
    name: string;
    nodename: string;
  }
  const generateMetricsUrl = (podDetails: PodDetail[]) => {
    // 根据 pod 的数量定义不同的路径
    const paths: { [key: number]: string } = {
      1: "p07YRssIz/pod1",
      2: "Xq4ygssSk/pod2",
      3: "TiCsgsySz/pod3",
      4: "bXd8gsySk/pod4",
    };

    // 获取对应 pod 数量的路径
    const path = paths[podDetails.length] || paths[1]; // 使用 pod1 的路径作为默认值

    const base = `http://192.168.5.60:31121/d/${path}?orgId=1`;

    // 生成 pod 名称和节点名称的参数
    const podParams = podDetails
      .map(
        (pod, index) =>
          `var-pod${index + 1}=${encodeURIComponent(pod.name)}&var-node${index + 1}=${encodeURIComponent(pod.nodename)}`,
      )
      .join("&");

    // 生成 GPU 参数
    const gpuParams = podDetails
      .map((_, index) => `var-pod${index + 1}_gpu=All`)
      .join("&");

    // 组合所有参数
    const queryParams = `${podParams}&${gpuParams}`;
    return `${base}&${queryParams}`;
  };

  return (
    <>
      <Card className="col-span-3">
        <CardContent className="flex items-center justify-between bg-muted/50 p-6">
          <div className="flex flex-row items-center gap-2">
            <h1 className="text-xl font-semibold capitalize text-foreground">
              {data.name}
            </h1>
          </div>
          <div className="flex items-center gap-2">
            <Select onValueChange={(value) => handleIntervalChange(value)}>
              <SelectTrigger className="w-[180px] bg-background">
                <SelectValue placeholder="Update Every 5s" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="5">Update Every 5s</SelectItem>
                <SelectItem value="10">Update Every 10s</SelectItem>
                <SelectItem value="20">Update Every 20s</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              size="icon"
              onClick={() =>
                void queryClient.invalidateQueries({
                  queryKey: ["job", "detail", jobName],
                })
              }
            >
              <RefreshCcwIcon className="h-3.5 w-3.5" />
            </Button>
          </div>
        </CardContent>
        <Separator />
        <CardContent className="flex flex-col gap-4 p-7">
          <div
            className="grid grid-cols-7 gap-12"
            style={{ gridTemplateColumns: "repeat(7, 1fr)" }}
          >
            <div>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline">{data.jobName}</Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Job Name</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
            <div>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline">{data.createdAt}</Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Created At</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
            <div>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline">{data.username}</Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Username</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
            <div>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline">{data.namespace}</Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Namespace</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
            <div>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline">{data.runtime}</Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Runtime</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
            <div>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline">{data.status}</Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Status</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
            <div>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline">{data.retry}</Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Retry</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Separator orientation="vertical" />
              {/* <Button className="text-blue-500" size="sm" variant="ghost">
                Go to Multi-Metrics Page
              </Button> */}
              <Button
                className="text-blue-500"
                size="sm"
                variant="ghost"
                onClick={() => {
                  const url = generateMetricsUrl(data.podDetails);
                  window.open(url, "_blank");
                }}
              >
                Go to Multi-Metrics Page
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
      <Card className="col-span-3 p-7">
        <Table className="w-full">
          <TableHeader>
            <TableRow>
              <TableHead>No.</TableHead>
              <TableHead>Container ID</TableHead>
              <TableHead>Assigned Node</TableHead>
              <TableHead>IP</TableHead>
              <TableHead>Ports</TableHead>
              <TableHead>Resources</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.podDetails &&
              Array.isArray(data.podDetails) &&
              data.podDetails.map((pod, index) => (
                <TableRow key={index}>
                  <TableCell>{index}</TableCell>
                  <TableCell className="font-medium">
                    {pod.name ? (
                      <a
                        href={`http://192.168.5.60:31121/d/MhnFUFLSz/pod_memory?orgId=1&var-node_name=${pod.nodename}&var-pod_name=${pod.name}`}
                        className="hover:underline"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {pod.name}
                      </a>
                    ) : (
                      "---"
                    )}
                  </TableCell>
                  <TableCell>
                    {pod.nodename ? (
                      <a
                        href={`http://192.168.5.60:31121/d/Oxed_c6Wz1/k8s-vgpu-scheduler-dashboard?orgId=1&var-node_name=${pod.nodename}`}
                        className="hover:underline"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {pod.nodename}
                      </a>
                    ) : (
                      "---"
                    )}
                  </TableCell>
                  <TableCell>{pod.ip || "---"}</TableCell>
                  <TableCell>{pod.port || "---"}</TableCell>
                  <TableCell>
                    {pod.resource &&
                      Object.entries(handleResourceData(pod.resource)).map(
                        ([key, value]) => (
                          <Badge
                            key={key}
                            variant="secondary"
                            className="gap-2 font-medium"
                          >
                            {" "}
                            {key}: {String(value)}{" "}
                          </Badge>
                        ),
                      )}
                  </TableCell>
                  <TableCell>{pod.status || "---"}</TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </Card>
    </>
  );
};

const JobDetail = () => {
  const { id } = useParams<string>();
  return (
    <>
      <div>Job Detail: {id}</div>
      <JupyterDetail />
    </>
  );
};
export default JobDetail;
