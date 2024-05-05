import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";
import { RefreshCcwIcon } from "lucide-react";
import { useEffect } from "react";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { InfoCircledIcon } from "@radix-ui/react-icons";
import { Separator } from "@/components/ui/separator";
import { Alert, AlertDescription } from "@/components/ui/alert";

const JupyterDetail = () => {
  const { id } = useParams();
  const setBreadcrumb = useBreadcrumb();

  useEffect(() => {
    setBreadcrumb([{ title: "任务详情" }]);
  }, [setBreadcrumb]);

  return (
    <>
      <Alert className="md:col-span-3" variant={"destructive"}>
        <AlertDescription>
          参考
          https://openpai.readthedocs.io/zh-cn/latest/manual/cluster-user/how-to-debug-jobs.html
          , 完善一下各个功能的入口。
        </AlertDescription>
      </Alert>
      <Card className="col-span-3">
        <CardContent className="flex items-center justify-between bg-muted/50 p-6">
          <div className="flex flex-row items-center gap-2">
            <h1 className="text-xl font-semibold capitalize text-foreground">
              {id}
            </h1>
            <InfoCircledIcon className="h-4 w-4 text-gray-500" />
          </div>
          <div className="flex items-center gap-2">
            <Select>
              <SelectTrigger className="w-[180px] bg-background">
                <SelectValue placeholder="Update Every 5s" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="5s">Update Every 5s</SelectItem>
                <SelectItem value="10s">Update Every 10s</SelectItem>
                <SelectItem value="20s">Update Every 20s</SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline" size="icon">
              <RefreshCcwIcon className="h-3.5 w-3.5" />
            </Button>
          </div>
        </CardContent>
        <Separator />
        <CardContent className="flex flex-col gap-4 p-6">
          <div className="flex space-x-4">
            <Badge variant="secondary">Running</Badge>
            <span className="text-sm text-gray-600">2d:3h:14m:14s</span>
            <span className="text-sm text-gray-600">admin</span>
            <Badge variant="default">default</Badge>
            <span className="text-sm text-gray-600">2d:14h:2m</span>
            <span className="text-sm text-gray-600">0</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Button size="sm" variant="ghost">
                View Job Config
              </Button>
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
              <Button variant="default">Clone</Button>
              <Button variant="destructive">Stop</Button>
            </div>
          </div>
        </CardContent>
      </Card>
      <Card className="col-span-3 p-6">
        <Table className="w-full">
          <TableHeader>
            <TableRow>
              <TableHead>No.</TableHead>
              <TableHead>Container ID</TableHead>
              <TableHead>IP</TableHead>
              <TableHead>Ports</TableHead>
              <TableHead>GPUs</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Info</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow>
              <TableCell>0</TableCell>
              <TableCell className="font-medium">
                cfcd208495d565ef66e7dff9f98764da
              </TableCell>
              <TableCell>---</TableCell>
              <TableCell>ssh: 25592 http: ---</TableCell>
              <TableCell>1</TableCell>
              <TableCell>Running</TableCell>
              <TableCell>
                <div className="flex space-x-2">
                  <Badge variant="default">Stdout</Badge>
                  <Badge variant="secondary">Stderr</Badge>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </Card>
    </>
  );
};

export default JupyterDetail;
