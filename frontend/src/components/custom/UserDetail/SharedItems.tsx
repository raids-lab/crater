// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

const items = [
  { id: 1, name: "图像分类模型", type: "模型", downloads: 1200 },
  { id: 2, name: "自然语言处理数据集", type: "数据集", downloads: 850 },
  { id: 3, name: "目标检测模型", type: "模型", downloads: 2000 },
];

export default function SharedItems() {
  const { t } = useTranslation();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("sharedItems.title")}</CardTitle>
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
                <p className="mt-2">
                  {t("sharedItems.downloadCount")}: {item.downloads}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
