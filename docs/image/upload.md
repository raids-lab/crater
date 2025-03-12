# 上传本地镜像

本文档将指导您如何将本地制作的镜像上传到平台 Harbor 仓库，以便在平台中创建作业时使用。

## 1. 获取 Harbor 访问凭据

### 1.1 通过平台获得访问凭据

首先，您需要在平台获取 Harbor 的访问凭据。您可以通过左侧菜单栏来到“镜像管理”中的“镜像制作”页面，在该页面中，您可以使用下图所示的“获取初始凭据”按钮，来获取或重置您 Harbor 仓库的访问凭据。您的 Harbor 账号将独立于您的平台账号。

![获取初始凭据](img/imageupload-getkey.png)

点击该按钮后，会弹出如图所示的对话框，包含您 Harbor 的用户名和密码，请您复制并妥善保管。

![保存初始凭据](img/imageupload-key.png)
### 1.2 登录 Harbor

在获取到 Harbor 的访问凭据后，您可以在您待上传镜像所在的主机上，使用如下命令登录 Harbor：

```bash
docker login <Harbor 仓库地址> -u <用户名> -p <密码>
```

其中，`<用户名>` 和 `<密码>` 分别为您在平台获取到的 Harbor 的用户名和密码，`<Harbor 仓库地址>` 为 `gpu-harbor.act.buaa.edu.cn`。

登陆成功后，您可以在命令行输出中看到 `Login Succeeded` 的提示，同时，在通常情况下，您的登录凭据将会被保存在 `~/.docker/config.json` 文件中，之后您再次上传镜像时，无需重新登录。

### 1.3 （可选）重置或修改您的密码

如果您忘记了您的密码，您可以再次点击上述“获取初始凭据”按钮，此时平台将会重置您的 Harbor 密码，并弹出新的对话框，包含用户名和新的密码，请您复制并妥善保管。

如果您希望修改您的密码，则您需要使用浏览器打开 Harbor 的 Web UI（可直接通过上述 Harbor 仓库地址访问），使用您现有的用户名和密码登录，然后单击右上角您的用户名，并点击“修改密码”以修改您的密码。

![修改密码](img/imageupload-passwd.png)

## 2. 上传本地镜像到 Harbor 仓库

在进入本章节之前，请确认您已经成功的在您镜像所在的主机上登陆了 Harbor，并看到 `Login Succeeded` 字样，如果没有，请您返回本文档“登录 Harbor”一节。

### 2.1 标记本地镜像

在推送您的镜像之前，您需要将本地镜像标记为符合 Harbor 仓库的命名规范。例如：

```bash
docker tag local-image:tag harbor.example.com/project-name/repository-name:tag
```

- `local-image:tag`: 您本地的镜像名称和标签。
- `harbor.example.com/project-name/repository-name:tag`: Harbor 仓库的完整路径，包括项目名称、仓库名称和标签。

其中，您的项目名称是 `user-{ACT 用户名}`，此外，您的 Harbor 账号具有该项目的项目管理员身份。

一个符合 Harbor 命名规范的完整镜像名称示例如下：

```bash
gpu-harbor.act.buaa.edu.cn/user-liuyizhou/nvidia-pytorch:24.12-v1.2.2
```

### 2.2 推送镜像到 Harbor

使用 `docker push` 命令将标记好的镜像推送到 Harbor 仓库：

```bash
docker push harbor.example.com/project-name/repository-name:tag
```

推送成功后，您可以在 Harbor 仓库的 Web UI 中看到上传的镜像。

## 3. 导入镜像

当您成功地把镜像推送至 Harbor 仓库后，您还需要在平台中导入您的镜像。

在平台中，您可以通过左侧菜单栏来到“镜像管理”中的“镜像列表”界面，然后通过页面右上角的“导入镜像”按钮打开镜像导入表单，如图所示。

![导入镜像](img/imageupload-import.png)

在该表单中，您需要填写完整的镜像链接，即包含 Harbor 仓库地址、项目名称、仓库名称和标签的完整镜像链接，示例如图所示。

![导入镜像](img/imageupload-importsheet.png)

除此之外，您还需要填写一个简短的镜像描述，作为镜像的标题展示，以及一个镜像的目标任务类型。这二者可以在导入镜像成功后随时修改。

最后，点击“提交镜像”按钮，即可完成镜像的导入。此时您应该能够看到导入成功的提示，并能够在新建作业时看到您新导入的镜像。

## 4. 常见问题

### 4.1 Harbor 登录失败

- 确保用户名和密码正确，如果您忘记密码，可以通过上述“获取初始凭据”按钮重置。
- 检查 Harbor URL 是否正确，网络是否联通。

### 4.2 镜像推送失败

- 确保已成功登录。
- 确保镜像标记的路径正确且完整。
- 检查您是否有权限将镜像推送到指定的项目和仓库。
- 检查您的存储空间配额是否已满。
- 如果推送在进度条走完时卡住，并反复自动重试，请您联系平台管理员。
