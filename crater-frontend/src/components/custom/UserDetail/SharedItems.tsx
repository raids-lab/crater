import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

const items = [
  { id: 1, name: "图像分类模型", type: "模型", downloads: 1200 },
  { id: 2, name: "自然语言处理数据集", type: "数据集", downloads: 850 },
  { id: 3, name: "目标检测模型", type: "模型", downloads: 2000 },
];

export default function SharedItems() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>分享的模型和数据集</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {items.map((item) => (
            <Card key={item.id}>
              <CardHeader>
                <CardTitle className="text-lg">{item.name}</CardTitle>
              </CardHeader>
              <CardContent>
                <Badge>{item.type}</Badge>
                <p className="mt-2">下载次数: {item.downloads}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
