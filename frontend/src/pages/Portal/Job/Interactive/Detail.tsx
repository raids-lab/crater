import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import { RefreshCcwIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { PlayIcon, StopIcon } from "@radix-ui/react-icons";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui-custom/alert-dialog";
import { useNavigate } from "react-router-dom";
import {
  TableHead,
  TableRow,
  TableHeader,
  TableCell,
  TableBody,
  Table,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";
import {
  apiJupyterTokenGet,
  IJupyterDetail,
  JobPhase,
  apiJobDelete,
  apiJupyterGetDetail,
  apiJupyterLog,
  apiJupyterYaml,
  Logs,
} from "@/services/api/vcjob";
import LogSheet from "@/components/custom/LogSheet";
export interface Resource {
  [key: string]: string;
}

export const Component = () => {
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
    queryFn: () => apiJupyterGetDetail(jobName),
    select: (res) => res.data.data,
    refetchInterval: refetchInterval,
  });

  const handleIntervalChange = (newInterval: string) => {
    const intervalMs = parseInt(newInterval) * 1000; // Convert seconds to milliseconds
    setRefetchInterval(intervalMs);
  };

  const [logs, setLogs] = useState<Logs>({});
  const [yaml, setYaml] = useState("Loading...");

  const { data: fetchLogsData, isLoading: fetchLogsBool } = useQuery({
    queryKey: ["job", "logs", jobName],
    queryFn: () => apiJupyterLog(jobName),
    select: (res) => res.data.data.logs,
    refetchInterval: refetchInterval,
  });

  const { data: fetchYamlData, isLoading: fetchYamlBool } = useQuery({
    queryKey: ["job", "yaml", jobName],
    queryFn: () => apiJupyterYaml(jobName),
    select: (res) => res.data.data,
    refetchInterval: refetchInterval,
  });

  const navigate = useNavigate();
  const { mutate: deleteJTask } = useMutation({
    mutationFn: (jobName: string) => apiJobDelete(jobName),
    onSuccess: () => {
      navigate("/jupyter");
      toast.success("作业已删除");
    },
  });

  const { mutate: getPortToken } = useMutation({
    mutationFn: (jobName: string) => apiJupyterTokenGet(jobName),
    onSuccess: (_, jobName) => {
      window.open(`/job/jupyter/${jobName}`);
    },
  });

  const { mutate: goToTensorboardPage } = useMutation({
    mutationFn: (jobName: string) => apiJupyterTokenGet(jobName),

    onSuccess: (_, jobName) => {
      window.open(`https://crater.act.buaa.edu.cn/tensorboard/${jobName}`);
    },
  });

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

  useEffect(() => {
    if (!fetchLogsBool && fetchLogsData) {
      setLogs(fetchLogsData);
    }
    if (!fetchYamlBool && fetchYamlData) {
      setYaml(fetchYamlData);
    }
  }, [fetchLogsBool, fetchLogsData, fetchYamlBool, fetchYamlData]);
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
          <Table className="w-full">
            <TableHeader>
              <TableRow>
                <TableHead>Job Name</TableHead>
                <TableHead>Created At</TableHead>
                <TableHead>Username</TableHead>
                <TableHead>Namespace</TableHead>
                <TableHead>Runtime</TableHead>
                <TableHead>Retry</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>{data.jobName}</TableCell>
                <TableHead>{data.createdAt}</TableHead>
                <TableHead>{data.username}</TableHead>
                <TableHead>{data.namespace}</TableHead>
                <TableHead>{data.runtime}</TableHead>
                <TableHead>{data.status}</TableHead>
                <TableHead>{data.retry}</TableHead>
              </TableRow>
            </TableBody>
          </Table>
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <LogSheet log={yaml} title={jobName + " Job Yaml"}>
                <Button size="sm" variant="ghost">
                  View Job Yaml
                </Button>
              </LogSheet>
              <Separator orientation="vertical" />
              <Button size="sm" variant="ghost">
                View Ekdi Diagnostics
              </Button>
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
              <Separator orientation="vertical" />
              {data.useTensorBoard && (
                <Button
                  className="text-blue-500"
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    toast.info("即将跳转至 TensorBoard 页面");
                    setTimeout(() => {
                      goToTensorboardPage(jobName);
                    }, 500);
                  }}
                  disabled={data.status !== JobPhase.Running}
                >
                  Go to TensorBoard Page
                </Button>
              )}
            </div>
            <div className="flex space-x-2">
              <Button
                variant="outline"
                className="h-8 w-8 p-0 hover:text-sky-700"
                title="运行 Jupyter Lab"
                onClick={() => {
                  toast.info("即将跳转至 Jupyter 页面");
                  setTimeout(() => {
                    getPortToken(jobName);
                  }, 500);
                }}
                disabled={data.status !== JobPhase.Running}
              >
                <PlayIcon />
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button
                      variant="outline"
                      className="h-8 w-8 p-0 hover:text-red-700"
                      title="终止 Jupyter Lab"
                    >
                      <StopIcon />
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除作业</AlertDialogTitle>
                    <AlertDialogDescription>
                      作业「{data.name}
                      」将停止，请确认已经保存好所需数据。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        // check if browser support clipboard
                        deleteJTask(jobName);
                      }}
                    >
                      删除
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
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
              <TableHead>Info</TableHead>
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
                  <TableCell>
                    <div className="flex space-x-2">
                      <LogSheet log={logs[pod.name]} title={pod.name}>
                        <Button variant="default">Stdout/Stderr</Button>
                      </LogSheet>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </Card>
    </>
  );
};
