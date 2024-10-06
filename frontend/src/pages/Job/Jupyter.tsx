import type { FC } from "react";
import { useEffect, useMemo } from "react";
import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { apiJobLogs, apiJupyterTokenGet } from "@/services/api/vcjob";
import CraterIcon from "@/components/icon/CraterIcon";
import { Button } from "@/components/ui/button";
import CodeSheet from "@/components/codeblock/CodeSheet";

const Jupyter: FC = () => {
  // get param from url
  const { id } = useParams();

  const { data: taskLogs } = useQuery({
    queryKey: ["jupyter", "tasklog", id],
    queryFn: () => apiJobLogs(id ?? ""),
    select: (res) => {
      const logs = res.data.data.logs;
      const firstKey = Object.keys(logs)[0];
      const firstValue = logs[firstKey];
      return firstValue;
    },
    enabled: !!id,
  });

  const { data: jupyterInfo } = useQuery({
    queryKey: ["jupyter", id],
    queryFn: () => apiJupyterTokenGet(id ?? "0"),
    select: (res) => res.data.data,
    enabled: !!id,
  });

  const url = useMemo(() => {
    if (jupyterInfo) {
      return `https://crater.act.buaa.edu.cn/jupyter/${jupyterInfo.baseURL}?token=${jupyterInfo.token}`;
    } else {
      return "";
    }
  }, [jupyterInfo]);

  // set jupyter notebook icon as current page icon
  // icon url: `https://crater.act.buaa.edu.cn/jupyter/${jupyterInfo.baseURL}/static/favicons/favicon.ico`
  // set title to jupyter base url
  useEffect(() => {
    if (jupyterInfo?.baseURL) {
      const link = document.querySelector(
        "link[rel='website icon']",
      ) as HTMLLinkElement;
      if (link) {
        link.href = `https://crater.act.buaa.edu.cn/jupyter/${jupyterInfo.baseURL}/static/favicons/favicon.ico`;
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
      <CodeSheet code={taskLogs} title={id}>
        <Button
          className="absolute bottom-10 right-4 h-10 w-10 rounded-full hover:bg-transparent"
          size="icon"
          variant={"ghost"}
          title="Jupyter 日志"
        >
          <CraterIcon />
        </Button>
      </CodeSheet>
    </div>
  );
};

export default Jupyter;
