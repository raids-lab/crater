import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import { RefreshCcwIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
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
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
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
import { InfoCircledIcon } from "@radix-ui/react-icons";
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";
import {
  apiJupyterTokenGet,
  IJupyterDetail,
  JobPhase,
  apiJupyterDelete,
  apiJupyterGetDetail,
  apiJupyterLog,
  apiJupyterYaml,
} from "@/services/api/jupyterTask";

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
  };
  const { id } = useParams<string>();
  const jobName = "" + id;
  const setBreadcrumb = useBreadcrumb();
  const [data, setData] = useState<IJupyterDetail>(initial);
  const [refetchInterval, setRefetchInterval] = useState(5000); // Manage interval state

  const { data: taskList, isLoading } = useQuery({
    queryKey: ["jupyter", "detail", jobName],
    queryFn: () => apiJupyterGetDetail(jobName),
    select: (res) => res.data.data,
    refetchInterval: refetchInterval,
  });

  const handleIntervalChange = (newInterval: string) => {
    const intervalMs = parseInt(newInterval) * 1000; // Convert seconds to milliseconds
    setRefetchInterval(intervalMs);
  };

  const [showLog, setShowLog] = useState(false);
  const [showYaml, setShowYaml] = useState(false);
  const [isFetchingLogs, setFetchingLogs] = useState(false);
  const [isFetchingYaml, setFetchingYaml] = useState(false);
  const [logs, setLogs] = useState("");
  const [yaml, setYaml] = useState("");

  const { mutate: fetchLogs } = useMutation({
    mutationFn: apiJupyterLog,
    onSuccess: (res) => {
      setLogs(res.data.data);
      setShowLog(true);
      setFetchingLogs(true);
    },
  });

  const { mutate: fetchYaml } = useMutation({
    mutationFn: apiJupyterYaml,
    onSuccess: (res) => {
      setYaml(res.data.data);
      setShowYaml(true);
      setFetchingYaml(true);
    },
  });

  const handleFetchLogs = (jobName: string) => {
    setFetchingLogs(false);
    fetchLogs(jobName);
  };

  const handleFetchYaml = (jobName: string) => {
    setFetchingYaml(false);
    fetchYaml(jobName);
  };

  const navigate = useNavigate();
  const { mutate: deleteJTask } = useMutation({
    mutationFn: (jobName: string) => apiJupyterDelete(jobName),
    onSuccess: () => {
      navigate("");
      toast.success("任务已删除");
    },
  });

  const { mutate: getPortToken } = useMutation({
    mutationFn: (jobName: string) => apiJupyterTokenGet(jobName),
    onSuccess: (_, jobName) => {
      window.open(`/job/jupyter/${jobName}`);
    },
  });

  useEffect(() => {
    setBreadcrumb([{ title: "任务详情" }]);
  }, [setBreadcrumb]);

  useEffect(() => {
    if (!isLoading && taskList) {
      setData(taskList);
    }
  }, [taskList, isLoading, taskList?.runtime]);

  return (
    <>
      <Card className="col-span-3">
        <CardContent className="flex items-center justify-between bg-muted/50 p-6">
          <div className="flex flex-row items-center gap-2">
            <h1 className="text-xl font-semibold capitalize text-foreground">
              {data.name}
            </h1>
            <InfoCircledIcon className="h-4 w-4 text-gray-500" />
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
            <Button variant="outline" size="icon">
              <RefreshCcwIcon className="h-3.5 w-3.5" />
            </Button>
          </div>
        </CardContent>
        <Separator />
        <CardContent className="flex flex-col gap-4 p-6">
          <div className="grid grid-cols-6 gap-4">
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
              <Dialog open={showYaml} onOpenChange={setShowYaml}>
                <DialogTrigger asChild>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => handleFetchYaml(jobName)}
                  >
                    View Job Config
                  </Button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-[800px]">
                  <DialogHeader>
                    <DialogTitle>Job Yaml</DialogTitle>
                  </DialogHeader>
                  <Textarea
                    className="h-72 rounded-md border"
                    placeholder={isFetchingYaml ? yaml : "Loading..."}
                  />
                  <DialogFooter>
                    <Button type="submit" onClick={() => setShowYaml(false)}>
                      Close
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
              <Separator orientation="vertical" />
              <Button size="sm" variant="ghost">
                View Ekdi Diagnostics
              </Button>
              <Separator orientation="vertical" />
              <Button className="text-blue-500" size="sm" variant="ghost">
                Go to Multi-Metrics Page
              </Button>
              <Separator orientation="vertical" />
              <Button className="text-blue-500" size="sm" variant="ghost">
                Go to TensorBoard Page
              </Button>
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
                    <AlertDialogTitle>删除任务</AlertDialogTitle>
                    <AlertDialogDescription>
                      任务「{data.name}
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
                    {pod.name || "---"}
                  </TableCell>
                  <TableCell>{pod.nodename || "---"}</TableCell>
                  <TableCell>{pod.ip || "---"}</TableCell>
                  <TableCell>{pod.port || "---"}</TableCell>
                  <TableCell>{pod.resource || "---"}</TableCell>
                  <TableCell>{pod.status || "---"}</TableCell>
                  <TableCell>
                    <div className="flex space-x-2">
                      <Dialog open={showLog} onOpenChange={setShowLog}>
                        <DialogTrigger asChild>
                          <Badge
                            variant="default"
                            onClick={() => handleFetchLogs(jobName)}
                          >
                            Stdout/Stderr
                          </Badge>
                        </DialogTrigger>
                        <DialogContent className="sm:max-w-[800px]">
                          <DialogHeader>
                            <DialogTitle>Stdout/Stderr logs</DialogTitle>
                          </DialogHeader>
                          <Textarea
                            className="h-72 rounded-md border"
                            placeholder={isFetchingLogs ? logs : "Loading..."}
                          />
                          <DialogFooter>
                            <Button
                              type="submit"
                              onClick={() => setShowLog(false)}
                            >
                              Close
                            </Button>
                          </DialogFooter>
                        </DialogContent>
                      </Dialog>
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

export default JupyterDetail;
