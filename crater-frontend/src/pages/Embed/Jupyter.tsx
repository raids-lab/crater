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
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useAtomValue } from "jotai";
import { urlHostAtom } from "@/utils/store/config";

const Jupyter: FC = () => {
  // get param from url
  const { id } = useParams();
  const [namespacedName, setNamespacedName] = useState<NamespacedName>();
  const [isSnapshotOpen, setIsSnapshotOpen] = useState(false);
  const host = useAtomValue(urlHostAtom);

  const { data: jupyterInfo } = useQuery({
    queryKey: ["jupyter", id],
    queryFn: () => apiJupyterTokenGet(id ?? "0"),
    select: (res) => res.data.data,
    enabled: !!id,
  });

  const url = useMemo(() => {
    if (jupyterInfo && host) {
      return `https://${host}/ingress/${jupyterInfo.baseURL}?token=${jupyterInfo.token}`;
    }
    return "";
  }, [jupyterInfo, host]);

  // set jupyter notebook icon as current page icon
  // set title to jupyter base url
  useEffect(() => {
    if (jupyterInfo?.baseURL && host) {
      const link = document.querySelector(
        "link[rel='website icon']",
      ) as HTMLLinkElement;
      if (link) {
        link.href = `https://${host}/ingress/${jupyterInfo.baseURL}/static/favicons/favicon.ico`;
        link.type = "image/x-icon";
      }
      document.title = jupyterInfo.baseURL;
    }
  }, [jupyterInfo, host]);

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
        className="absolute top-0 right-0 bottom-0 left-0 h-screen w-screen"
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
      <AlertDialog open={isSnapshotOpen} onOpenChange={setIsSnapshotOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>保存镜像</AlertDialogTitle>
            <AlertDialogDescription>
              保存当前作业的镜像，保存期间作业将被暂停
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => snapshot(id ?? "")}>
              保存
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default Jupyter;
