import { useQuery } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { Button } from "@/components/ui/button";
import { apiUserGetQuota } from "@/services/api/imagepack";
import { BoxIcon, HardDriveIcon, KeyIcon, UserRoundIcon } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import LoadingCircleIcon from "@/components/icon/LoadingCircleIcon";
import { UserHarborCredentialsDialog } from "./UserHarborCredentialsDialog";

interface ProjectDetailProps {
  successImageNumber: number;
}

export const ProjectDetail: FC<ProjectDetailProps> = ({
  successImageNumber,
}) => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const quotaQuery = useQuery({
    queryKey: ["imagepack", "quota"],
    queryFn: () => apiUserGetQuota(),
    select: (res) => res.data.data,
  });
  return (
    <>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="shadow-xs">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              <BoxIcon className="text-primary size-5" />
              镜像总数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {quotaQuery.isLoading ? (
                <LoadingCircleIcon />
              ) : (
                <>{successImageNumber}</>
              )}
            </p>
          </CardContent>
        </Card>
        <Card className="shadow-xs">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              <HardDriveIcon className="text-primary size-5" />
              存储用量
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {quotaQuery.isLoading ? (
                <LoadingCircleIcon />
              ) : (
                <>
                  {Number(quotaQuery.data?.used).toFixed(2)}GiB/
                  {Number(quotaQuery.data?.quota).toFixed(0)}GiB
                </>
              )}
            </p>
          </CardContent>
        </Card>

        <Card className="shadow-xs">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              <UserRoundIcon className="text-primary size-5" />
              仓库项目
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {quotaQuery.isLoading ? (
                <LoadingCircleIcon />
              ) : (
                <>{quotaQuery.data?.project}</>
              )}
            </p>
          </CardContent>
        </Card>

        <Card className="shadow-xs">
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              <KeyIcon className="text-primary size-5" />
              访问凭据
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Button
              variant="outline"
              onClick={() => setIsDialogOpen(true)}
              className="w-full"
            >
              获取初始凭据
            </Button>
          </CardContent>
        </Card>
      </div>

      <UserHarborCredentialsDialog
        isDialogOpen={isDialogOpen}
        setIsDialogOpen={setIsDialogOpen}
      />
    </>
  );
};
