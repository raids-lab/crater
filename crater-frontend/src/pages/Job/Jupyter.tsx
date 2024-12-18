import type { FC } from "react";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { apiJupyterSnapshot, apiJupyterTokenGet } from "@/services/api/vcjob";
import LogDialog from "@/components/codeblock/LogDialog";
import { NamespacedName } from "@/components/codeblock/PodContainerDialog";
import FloatingBall from "./FloatingBall";
import { toast } from "sonner";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

const Jupyter: FC = () => {
  // get param from url
  const { id } = useParams();
  const [namespacedName, setNamespacedName] = useState<NamespacedName>();
  const [isSnapshotOpen, setIsSnapshotOpen] = useState(false);

  const { data: jupyterInfo } = useQuery({
    queryKey: ["jupyter", id],
    queryFn: () => apiJupyterTokenGet(id ?? "0"),
    select: (res) => res.data.data,
    enabled: !!id,
  });

  const url = useMemo(() => {
    if (jupyterInfo) {
      return `https://${import.meta.env.VITE_HOST}/ingress/${jupyterInfo.baseURL}?token=${jupyterInfo.token}`;
    } else {
      return "";
    }
  }, [jupyterInfo]);

  // set jupyter notebook icon as current page icon
  // icon url: `https://${import.meta.env.VITE_HOST}/ingress/${jupyterInfo.baseURL}/static/favicons/favicon.ico`
  // set title to jupyter base url
  useEffect(() => {
    if (jupyterInfo?.baseURL) {
      const link = document.querySelector(
        "link[rel='website icon']",
      ) as HTMLLinkElement;
      if (link) {
        link.href = `https://${import.meta.env.VITE_HOST}/ingress/${jupyterInfo.baseURL}/static/favicons/favicon.ico`;
        link.type = "image/x-icon";
      }
      document.title = jupyterInfo.baseURL;
    }
  }, [jupyterInfo]);

  const { mutate: snapshot } = useMutation({
    mutationFn: (jobName: string) => apiJupyterSnapshot(jobName),
    onSuccess: () => {
      toast.success("已提交快照");
      window.open("/portal/image/createimage", "_blank");
    },
  });

  // drag the floating ball to show log dialog
  const [isDragging, setIsDragging] = useState(false);

  return (
    <div className="relative h-screen w-screen">
      <iframe
        title="jupyter notebook"
        src={url}
        className="absolute bottom-0 left-0 right-0 top-0 h-screen w-screen"
      />
      {/* Transparent overlay */}
      {isDragging && (
        <div className="fixed inset-0 z-50" style={{ cursor: "move" }} />
      )}
      <FloatingBall
        jobName={id ?? ""}
        setIsDragging={setIsDragging}
        handleShowLog={() =>
          jupyterInfo &&
          setNamespacedName({
            name: jupyterInfo.podName,
            namespace: jupyterInfo.namespace,
          })
        }
        handleSnapshot={() => setIsSnapshotOpen(true)}
      />
      <LogDialog
        namespacedName={namespacedName}
        setNamespacedName={setNamespacedName}
      />
      <Dialog open={isSnapshotOpen} onOpenChange={setIsSnapshotOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>保存镜像</DialogTitle>
            <DialogDescription>
              保存当前作业的镜像，保存期间作业将被暂停
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline">取消</Button>
            <DialogClose>
              <Button
                type="submit"
                variant="default"
                onClick={() => snapshot(id ?? "")}
              >
                确认
              </Button>
            </DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Jupyter;
