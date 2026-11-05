export const platformUserGuideKnowledge = `# Crater 用户侧平台知识

## 知识边界
- 以 Crater 官方用户文档和当前用户端界面为依据，这些是稳定的使用知识，不是对当前集群状态的实时查询。
- 部署可能由管理员关闭部分入口或限制权限；若用户所见界面与文档不同，应以当前界面和管理员配置为准。

## 平台定位
- Crater 统一管理集群的计算与存储资源，用户可以创建交互式作业、提交批处理作业、管理镜像，以及使用文件、数据集和模型。
- “平台概览”展示资源概况；“集群监控”包含 GPU、空闲资源和网络监控；“个人仪表盘”展示用户相关信息。

## 作业类型选择
- Jupyter Lab 适合 Notebook、数据探索、交互调试和需要终端的持续开发。
- Web IDE 适合通过浏览器使用 IDE 进行代码编辑和调试。
- 自定义单机批处理作业适合已经调试完成、可按固定命令一次性运行的训练、推理或数据处理脚本。
- PyTorch DDP 和 TensorFlow PS 用于明确需要对应框架的分布式作业；不熟悉分布式训练时，不应默认推荐。
- 不确定类型时：需要反复操作、Notebook 或可视化调试选 Jupyter；需要浏览器 IDE 选 Web IDE；只需运行确定命令并等待结果选自定义单机作业。

## 创建作业的关键配置
- 通常需要设置作业名称、CPU、内存、GPU 数量与型号、容器镜像、数据挂载、环境变量，以及命令或工作目录。
- 镜像的 CPU 架构必须与可调度节点匹配，例如 amd64 镜像不能直接用于只支持 arm64 的节点。
- 在不知道精确资源需求时，建议先用较小配置做短时验证，再根据实际使用调整；不应凭空承诺某种 GPU 当前可用。
- 批处理作业的启动命令应先在交互式环境中验证。命令中的路径必须与挂载路径一致。
- 已有作业可以通过“克隆”复用原配置，然后在新建表单中调整。作业模板可用于共享和快速复现作业启动方式。

## 作业创建后
- 在“我的作业”中分别查看自定义作业和交互式作业，点击名称进入详情。
- 作业详情可展示基本信息、YAML、资源监控、交互式入口、Web Terminal、外部访问、日志与诊断等，具体入口取决于作业类型和平台配置。
- 停止作业会释放所占资源。停止或删除前应先确认需要的代码、模型和结果已保存到持久化挂载中。

## 文件与持久化
- “文件管理”可访问用户空间、账户空间和公共空间；账户空间只在用户已加入并选择账户时出现。
- 公共空间和账户空间的上传、删除等写操作可能需要额外权限。
- 新建作业的“数据挂载”可选择有权访问的文件夹、数据集、模型或共享文件，并指定容器内路径。
- Jupyter 作业通常会自动挂载用户空间；其他作业中的实际路径以创建表单里的挂载路径为准。
- 容器自身文件系统中未挂载的修改通常不是持久化数据。代码、训练结果和数据应写入挂载目录，不要只留在容器内部。

## 数据集、模型与共享文件
- 数据集和模型都是指向共享存储位置的只读资源，可以从 ModelScope 或 HuggingFace 下载，也可以把已上传的文件夹登记为数据集或模型。
- 作业应优先挂载平台已有的数据集或模型，而不是在每个容器中重复下载。
- 共享文件用于将用户可读的文件分享给用户或账户，接收方可按授权读写；它与只读的数据集、模型用途不同。
- 公共模型或数据集可能被多个用户复用。复用已有资源通常不会再创建下载作业。
- 下载额度可限制同时等待/下载的任务数和滚动时间窗口内完成数。暂停会保留部分文件，恢复时尽可能断点续传；只有 Ready 状态才表示文件完整可用。

## 镜像
- “镜像列表”包含自己制作或导入的镜像、公开镜像和被分享的镜像，可按名称、描述或标签检索。
- 镜像制作支持 Python+CUDA 表单、基于现有镜像、Dockerfile 和 Envd 等方式。不需要高度自定义时，优先使用结构化表单或现有镜像。
- 本地已有镜像可先推送到 Harbor，再在“镜像列表”中导入；仅推送到 Harbor 不代表已登记为 Crater 可选镜像。
- 在 Jupyter 容器中临时安装的包会随容器停止而丢失。可使用镜像保存功能，但大镜像保存容易受网络影响，官方文档更推荐通过 Dockerfile 或低代码表单制作可复现镜像。
- 镜像构建详情中可查看构建日志以及对应的 Dockerfile 或 Envd 配置。

## 常用页面
- 平台概览：/portal/overview
- 自定义作业：/portal/jobs/custom
- 交互式作业：/portal/jobs/inter
- 新建 Jupyter 作业：/portal/jobs/new/jupyter-job
- 新建 Web IDE 作业：/portal/jobs/new/webide-job
- 新建单机批处理作业：/portal/jobs/new/single-job
- 作业模板：/portal/templates
- 镜像制作：/portal/env/registry
- 镜像列表：/portal/env/images
- 数据集：/portal/data/datasets
- 模型：/portal/data/models
- 共享文件：/portal/data/blocks
- 文件管理：/portal/files
- 我的工单：/portal/more/orders
- 个人设置：/portal/more/user

## 新用户推荐路径
1. 明确目标：交互调试、Notebook、浏览器 IDE，还是一次性运行脚本。
2. 准备持久化内容：将代码上传到用户空间，先搜索平台是否已有所需数据集或模型。
3. 准备环境：先选可用的公共/共享镜像；需长期重复使用的依赖应制作新镜像，不要每次在容器中重新安装。
4. 先小规模验证：用 Jupyter 或 Web IDE 确认路径、依赖、GPU 可见性和启动命令。
5. 提交正式作业：对已稳定的脚本使用批处理作业，在详情页查看状态、日志和监控，并将结果写入挂载目录。

## 官方文档
- 用户指南：https://raids-lab.github.io/crater/zh/docs/user/
- 交互式作业：https://raids-lab.github.io/crater/zh/docs/user/quick-start/interactive/
- 自定义作业：https://raids-lab.github.io/crater/zh/docs/user/quick-start/batchprocess/
- 作业模板：https://raids-lab.github.io/crater/zh/docs/user/quick-start/jobtemplate/
- 镜像制作：https://raids-lab.github.io/crater/zh/docs/user/image/imageCreate/
- 镜像列表：https://raids-lab.github.io/crater/zh/docs/user/image/imageList/
- 文件系统：https://raids-lab.github.io/crater/zh/docs/user/file/file/
- 数据集：https://raids-lab.github.io/crater/zh/docs/user/file/dataset/
- 模型：https://raids-lab.github.io/crater/zh/docs/user/file/model/
- 下载额度：https://raids-lab.github.io/crater/zh/docs/user/file/download-quota/`;
