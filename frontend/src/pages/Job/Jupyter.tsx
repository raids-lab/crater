import type { FC } from "react";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { apiJupyterTokenGet } from "@/services/api/vcjob";
import LogDialog from "@/components/codeblock/LogDialog";
import { NamespacedName } from "@/components/codeblock/PodContainerDialog";
import FloatingBall from "./FloatingBall";

const Jupyter: FC = () => {
  // get param from url
  const { id } = useParams();
  const [namespacedName, setNamespacedName] = useState<NamespacedName>();

  const { data: jupyterInfo } = useQuery({
    queryKey: ["jupyter", id],
    queryFn: () => apiJupyterTokenGet(id ?? "0"),
    select: (res) => res.data.data,
    enabled: !!id,
  });

  const url = useMemo(() => {
    if (jupyterInfo) {
      return `https://crater.act.buaa.edu.cn/ingress/${jupyterInfo.baseURL}?token=${jupyterInfo.token}`;
    } else {
      return "";
    }
  }, [jupyterInfo]);

  // set jupyter notebook icon as current page icon
  // icon url: `https://crater.act.buaa.edu.cn/ingress/${jupyterInfo.baseURL}/static/favicons/favicon.ico`
  // set title to jupyter base url
  useEffect(() => {
    if (jupyterInfo?.baseURL) {
      const link = document.querySelector(
        "link[rel='website icon']",
      ) as HTMLLinkElement;
      if (link) {
        link.href = `https://crater.act.buaa.edu.cn/ingress/${jupyterInfo.baseURL}/static/favicons/favicon.ico`;
        link.type = "image/x-icon";
      }
      document.title = jupyterInfo.baseURL;
    }
  }, [jupyterInfo]);

  return (
    <div className="relative h-screen w-screen">
      <iframe
        title="jupyter notebook"
        src={url}
        className="absolute bottom-0 left-0 right-0 top-0 h-screen w-screen"
      />
      <FloatingBall
        handleShowLog={() =>
          jupyterInfo &&
          setNamespacedName({
            name: jupyterInfo.podName,
            namespace: jupyterInfo.namespace,
          })
        }
      />
      <LogDialog
        namespacedName={namespacedName}
        setNamespacedName={setNamespacedName}
      />
    </div>
  );
};

export default Jupyter;
