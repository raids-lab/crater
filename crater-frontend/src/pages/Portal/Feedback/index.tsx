import { SheetIcon } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";

export default function FeedBack() {
  return (
    <div>
      {/* 第二节：跳转至反馈在线表格 https://365.kdocs.cn/l/cuQ2oVGbNVIt */}
      <section>
        <Card>
          <CardHeader className="flex items-center space-x-2">
            <SheetIcon size={24} className="text-primary" />
            <CardTitle>平台反馈在线表格</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col items-center">
            <a
              href="https://365.kdocs.cn/l/cuQ2oVGbNVIt"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              点击跳转至反馈表格
            </a>
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
