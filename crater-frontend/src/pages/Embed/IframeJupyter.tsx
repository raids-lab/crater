// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from "react-i18next";
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
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

const Jupyter: FC = () => {
  const { t } = useTranslation();
  // get param from url
  const { id } = useParams();
  const [namespacedName, setNamespacedName] = useState<NamespacedName>();
  const [isSnapshotOpen, setIsSnapshotOpen] = useState(false);
  const [isDetailOpen, setIsDetailOpen] = useState(false);

  const { data: jupyterInfo } = useQuery({
    queryKey: ["jupyter", id],
    queryFn: () => apiJupyterTokenGet(id ?? "0"),
    select: (res) => res.data.data,
    enabled: !!id,
  });

  const url = useMemo(() => {
    if (jupyterInfo) {
      return `${jupyterInfo.fullURL}?token=${jupyterInfo.token}`;
    }
    return "";
  }, [jupyterInfo]);

  // set jupyter notebook icon as current page icon
  // set title to jupyter base url
  useEffect(() => {
    if (jupyterInfo?.baseURL && jupyterInfo.fullURL) {
      const link = document.querySelector(
        "link[rel='website icon']",
      ) as HTMLLinkElement;
      if (link) {
        link.href = `${jupyterInfo.fullURL}/static/favicons/favicon.ico`;
        link.type = "image/x-icon";
      }
      document.title = jupyterInfo.baseURL;
    }
  }, [jupyterInfo]);

  const { mutate: snapshot } = useMutation({
    mutationFn: (jobName: string) => apiJupyterSnapshot(jobName),
    onSuccess: () => {
      toast.success(t("jupyter.snapshot.success"));
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
        setIsDragging={setIsDragging}
        handleShowLog={() =>
          jupyterInfo &&
          setNamespacedName({
            name: jupyterInfo.podName,
            namespace: jupyterInfo.namespace,
          })
        }
        handleShowDetail={() => setIsDetailOpen(true)}
        handleSnapshot={() => setIsSnapshotOpen(true)}
      />
      <LogDialog
        namespacedName={namespacedName}
        setNamespacedName={setNamespacedName}
      />
      <AlertDialog open={isSnapshotOpen} onOpenChange={setIsSnapshotOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("jupyter.snapshot.title")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("jupyter.snapshot.description")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>
              {t("jupyter.snapshot.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction onClick={() => snapshot(id ?? "")}>
              {t("jupyter.snapshot.save")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <Sheet open={isDetailOpen} onOpenChange={setIsDetailOpen}>
        <SheetContent className="sm:max-w-6xl">
          <SheetHeader>
            <SheetTitle>{t("jupyter.detail.title")}</SheetTitle>
          </SheetHeader>
          <div className="h-[calc(100vh-6rem)] w-full px-4">
            <iframe
              title="grafana"
              src={`/portal/job/inter/${id}`}
              height={"100%"}
              width={"100%"}
            />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
};

export default Jupyter;
