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
    <Card className="h-[calc(100vh_-104px)] w-full">
      <CardHeader className="py-3"></CardHeader>
      <CardContent className="relative">
        <iframe
          src={`https://crater.act.buaa.edu.cn/dufs/share`}
          className="h-[calc(100vh_-152px)] w-full"
        />
      </CardContent>
    </Card>
  );
};
