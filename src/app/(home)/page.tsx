import { Card } from "fumadocs-ui/components/card";

export default function HomePage() {
  return (
    <main className="flex flex-1 flex-col justify-center text-center">
      <h1 className="mb-4 text-2xl font-bold">Crater 算力平台</h1>
      <div className="text-fd-muted-foreground">
        <div className="container flex flex-row items-center justify-center gap-4">
          <Card
            title="算力平台"
            description="点击访问"
            href="https://gpu.act.buaa.edu.cn/portal"
            className="text-sky-600"
          />
          <Card
            title="用户文档"
            description="点击阅读"
            href="/docs"
            className="text-orange-600"
          />
        </div>
      </div>
    </main>
  );
}
