import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import {
  ExternalLinkIcon,
  FileSlidersIcon,
  FileTextIcon,
  PieChartIcon,
  RefreshCcwIcon,
  SquareTerminalIcon,
  TrashIcon,
} from "lucide-react";
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
import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
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
  apiJobGetDetail,
  apiJupyterYaml,
} from "@/services/api/vcjob";
import CodeSheet from "@/components/codeblock/CodeSheet";
import JobPhaseLabel from "@/components/phase/JobPhaseLabel";
import { TableDate } from "@/components/custom/TableDate";
import { PodNamespacedName } from "@/components/codeblock/PodContainerDialog";
import LogDialog from "@/components/codeblock/LogDialog";
import TerminalDialog from "@/components/codeblock/TerminalDialog";
export interface Resource {
  [key: string]: string;
}

const initial: IJupyterDetail = {
  name: "",
  namespace: "",
  username: "",
  jobName: "",
  retry: "",
  queue: "",
  status: JobPhase.Unknown,
  createdAt: "",
  startedAt: "",
  runtime: "",
  podDetails: [],
  useTensorBoard: false,
};

const job_monitor = import.meta.env.VITE_GRAFANA_JOB_MONITOR;
// const pod_memory = import.meta.env.VITE_GRAFANA_POD_MEMORY;
// const k8s_vgpu_scheduler_dashboard = import.meta.env
// .VITE_GRAFANA_K8S_VGPU_SCHEDULER_DASHBOARD;

export const Component = () => {
  const { id } = useParams<string>();
  const jobName = "" + id;
  const setBreadcrumb = useBreadcrumb();
  const [data, setData] = useState<IJupyterDetail>(initial);
  const [refetchInterval, setRefetchInterval] = useState(5000); // Manage interval state
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [showLogPod, setShowLogPod] = useState<PodNamespacedName | undefined>();
  const [showTerminalPod, setShowTerminalPod] = useState<
    PodNamespacedName | undefined
  >();

  const { data: taskList, isLoading } = useQuery({
    queryKey: ["job", "detail", jobName],
    queryFn: () => apiJobGetDetail(jobName),
    select: (res) => res.data.data,
    refetchInterval: refetchInterval,
  });

  const handleIntervalChange = (newInterval: string) => {
    const intervalMs = parseInt(newInterval) * 1000; // Convert seconds to milliseconds
    setRefetchInterval(intervalMs);
  };

  const [yaml, setYaml] = useState("Loading...");

  const { data: fetchYamlData, isLoading: fetchYamlBool } = useQuery({
    queryKey: ["job", "yaml", jobName],
    queryFn: () => apiJupyterYaml(jobName),
    select: (res) => res.data.data,
    refetchInterval: refetchInterval,
  });

  const { mutate: getPortToken } = useMutation({
    mutationFn: (jobName: string) => apiJupyterTokenGet(jobName),
    onSuccess: (res) => {
      const data = res.data.data;
      window.open(`${data.baseURL}?token=${data.token}`);
    },
  });

  const { mutate: deleteJTask } = useMutation({
    mutationFn: () => apiJobDelete(jobName),
    onSuccess: () => {
      navigate("/jupyter");
      toast.success("作业已删除");
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
    if (!fetchYamlBool && fetchYamlData) {
      setYaml(fetchYamlData);
    }
  }, [fetchYamlBool, fetchYamlData]);

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
              <SelectTrigger className="h-8 w-[130px] bg-background">
                <SelectValue placeholder="每 5s 刷新" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="5">每 5s 刷新</SelectItem>
                <SelectItem value="10">每 10s 刷新</SelectItem>
                <SelectItem value="20">每 20s 刷新</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
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
                <TableHead className="text-xs">作业名称</TableHead>
                <TableHead className="text-xs">状态</TableHead>
                <TableHead className="text-xs">账户</TableHead>
                <TableHead className="text-xs">用户</TableHead>
                <TableHead className="text-xs">命名空间</TableHead>
                <TableHead className="text-xs">创建于</TableHead>
                <TableHead className="text-xs">持续时间</TableHead>
                <TableHead className="text-xs">重试次数</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>{data.jobName}</TableCell>
                <TableCell>
                  <JobPhaseLabel jobPhase={data.status} />
                </TableCell>
                <TableCell>{data.queue}</TableCell>
                <TableCell>{data.username}</TableCell>
                <TableCell>{data.namespace}</TableCell>
                <TableCell>
                  <TableDate date={data.createdAt} />
                </TableCell>
                <TableCell>{data.runtime}</TableCell>
                <TableCell>{data.retry}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
          <div className="mt-4 flex items-center justify-between">
            <div className="flex h-5 items-center space-x-2">
              <CodeSheet code={yaml} title={jobName + " Job Yaml"}>
                <Button variant="ghost" size="sm">
                  <FileSlidersIcon className="mr-2 h-4 w-4" />
                  作业 YAML
                </Button>
              </CodeSheet>
              <Separator orientation="vertical" />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  // const url = generateMetricsUrl(data.podDetails);
                  const url = `${job_monitor}?orgId=1&var-job=${data.jobName}`;
                  window.open(url, "_blank");
                }}
              >
                <PieChartIcon className="mr-2 h-4 w-4" />
                资源监控
              </Button>
              {data.useTensorBoard && (
                <>
                  <Separator orientation="vertical" />
                  <Button
                    className="text-primary"
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
                </>
              )}
            </div>
            <div className="flex flex-row gap-2">
              <Button
                size="sm"
                title="跳转至交互式页面"
                onClick={() => {
                  toast.info("即将跳转至交互式页面");
                  setTimeout(() => {
                    getPortToken(jobName);
                  }, 500);
                }}
              >
                <ExternalLinkIcon className="mr-2 h-4 w-4" />
                交互式页面
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button variant="destructive" size="sm" title="删除作业">
                      <TrashIcon className="mr-2 h-4 w-4" />
                      删除作业
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
                        deleteJTask();
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
              <TableHead className="text-xs">序号</TableHead>
              <TableHead className="text-xs">Pod 名称</TableHead>
              <TableHead className="text-xs">已分配节点</TableHead>
              <TableHead className="text-xs">IP</TableHead>
              <TableHead className="text-xs">端口</TableHead>
              <TableHead className="text-xs">资源</TableHead>
              <TableHead className="text-xs">状态</TableHead>
              <TableHead className="text-xs"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.podDetails &&
              Array.isArray(data.podDetails) &&
              data.podDetails.map((pod, index) => (
                <TableRow key={index}>
                  <TableCell>{index + 1}</TableCell>
                  <TableCell>
                    {pod.name ? (
                      <a
                        href={`http://8.141.83.224:31120/d/MhnFUFLSz/pod_monitor?orgId=1&refresh=5s&var-node_name=${pod.nodename}&var-pod_name=${pod.name}&var-gpu=All&from=now-15m&to=now`}
                        className="underline-offset-4 hover:underline"
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
                        href={`http://8.141.83.224:31120/d/Oxed_c6Wz1/node_monitor?orgId=1&var-node=${pod.nodename}&from=now-30m&to=now`}
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
                    {/* TODO: @wgz replace with <ResourceBadges resources={resources} /> */}
                    {pod.resource ? (
                      <div className="flex flex-row gap-1">
                        {Object.entries(handleResourceData(pod.resource)).map(
                          ([key, value]) => (
                            <Badge
                              key={key}
                              variant="secondary"
                              className="font-normal"
                            >
                              {key}: {String(value)}
                            </Badge>
                          ),
                        )}
                      </div>
                    ) : (
                      "---"
                    )}
                  </TableCell>
                  <TableCell>{pod.status || "---"}</TableCell>
                  <TableCell>
                    <div className="flex items-center justify-center space-x-1">
                      <Button
                        variant="outline"
                        size="icon"
                        className="h-8 w-8 text-primary hover:text-primary/90"
                        title="打开终端"
                        onClick={() => {
                          setShowTerminalPod({
                            namespace: data.namespace,
                            name: pod.name,
                          });
                        }}
                      >
                        <SquareTerminalIcon className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="icon"
                        className="h-8 w-8"
                        title="查看日志"
                        onClick={() => {
                          setShowLogPod({
                            namespace: data.namespace,
                            name: pod.name,
                          });
                        }}
                      >
                        <FileTextIcon className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </Card>
      <LogDialog
        namespacedName={showLogPod}
        setNamespacedName={setShowLogPod}
      />
      <TerminalDialog
        namespacedName={showTerminalPod}
        setNamespacedName={setShowTerminalPod}
      />
    </>
  );
};
