import type { FC } from "react";

import { Card, CardContent, CardHeader } from "@/components/ui/card";

export const Component: FC = () => {
  // get param from url

  // const userInfo = useRecoilValue(globalUserInfo);

  // const url = useMemo(() => {
  //   const user = userInfo.id;
  //   if (user === "huangjx") {
  //     return `http://192.168.5.67:32381`;
  //   } else if (jupyterToken) {
  //     return `https://crater.act.buaa.edu.cn/jupyter/${userInfo.id}-${id}?token=${jupyterToken}`;
  //   } else {
  //     return "";
  //   }
  // }, [userInfo]);

  return (
    <Card>
      <CardHeader className="py-3"></CardHeader>
      <CardContent
        style={{
          position: "relative",
          height: "900px",
          width: "1200px",
        }}
      >
        <iframe
          src={`https://crater.act.buaa.edu.cn/dufs/share`}
          style={{
            width: "1200px",
            height: "1000px",
            position: "absolute",
            top: "1%",
            left: "2%",
          }}
        />
      </CardContent>
    </Card>
  );
};

