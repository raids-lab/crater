import type { FC } from "react";
import { useMemo } from "react";
import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { apiJTaskGetPortToken } from "@/services/api/jupyterTask";
import { useRecoilValue } from "recoil";
import { globalUserInfo } from "@/utils/store";
import CraterIcon from "@/components/icon/CraterIcon";

const Jupyter: FC = () => {
  // get param from url
  const { id } = useParams();
  const userInfo = useRecoilValue(globalUserInfo);

  const jupyterInfo = useQuery({
    queryKey: ["jtask", id],
    queryFn: () => apiJTaskGetPortToken(parseInt(id ?? "0")),
    select: (res) => res.data.data,
    enabled: !!id,
  });

  const url = useMemo(() => {
    //   if (jupyterPort) {
    //     window.open(`http://192.168.5.60:${jupyterPort}?token=${jupyterToken}`);
    //   } else {
    //     window.open(
    //       `https://crater.act.buaa.edu.cn/jupyter/${userInfo.id}-${id}?token=${jupyterToken}`,
    //     );
    //   }
    const jupyterPort = jupyterInfo.data?.port;
    const jupyterToken = jupyterInfo.data?.token;
    if (jupyterPort) {
      return `http://192.168.5.60:${jupyterPort}?token=${jupyterToken}`;
    } else if (jupyterToken) {
      return `https://crater.act.buaa.edu.cn/jupyter/${userInfo.id}-${id}?token=${jupyterToken}`;
    } else {
      return "";
    }
  }, [jupyterInfo, id, userInfo]);

  return (
    <div className="relative h-screen w-screen">
      <iframe
        src={url}
        className="absolute bottom-0 left-0 right-0 top-0 h-screen w-screen"
      />
      <CraterIcon className="absolute bottom-10 right-4 h-10 w-10" />
    </div>
  );
};

export default Jupyter;
