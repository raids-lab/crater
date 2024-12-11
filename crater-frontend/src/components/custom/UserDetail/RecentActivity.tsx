import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";

const activities = [
  { id: 1, action: "上传了新的数据集", date: "2023-03-01" },
  { id: 2, action: "更新了图像分类模型", date: "2023-02-28" },
  { id: 3, action: "发表了新的研究论文", date: "2023-02-25" },
  { id: 4, action: "参与了在线研讨会", date: "2023-02-20" },
  { id: 5, action: "分享了代码仓库", date: "2023-02-15" },
];

export default function RecentActivity() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>近期活动</CardTitle>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[200px]">
          {activities.map((activity) => (
            <div key={activity.id} className="mb-4">
              <p className="font-semibold">{activity.action}</p>
              <p className="text-sm text-muted-foreground">{activity.date}</p>
            </div>
          ))}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
