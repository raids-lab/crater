---
title: VSCode를 Jupyter 컨테이너에 연결

description: 사용자가 Jupyter 작업을 생성한 후 VSCode를 통해 컨테이너 환경에 직접 연결할 수 있도록 지원하여 VSCode의 코드 자동 완성, 디버깅 기능 및 풍부한 플러그인 생태계를 활용함으로써 개발 효율성과 사용자 경험을 향상시킵니다.
---

# VSCode를 Jupyter 컨테이너에 연결

사용자가 Jupyter 작업을 생성한 후 VSCode를 통해 컨테이너 환경에 직접 연결할 수 있도록 지원하여 VSCode의 코드 자동 완성, 디버깅 기능 및 풍부한 플러그인 생태계를 활용함으로써 개발 효율성과 사용자 경험을 향상시킵니다.

## 로컬에서 공개키와 개인키 생성 확인

시작하기 전에 로컬에서 공개키와 개인키 파일이 이미 생성되어 있는지 확인해야 합니다. 일반적으로 `C:\Users\<사용자이름>\.ssh` 또는 `~/.ssh` 디렉터리에 저장됩니다. 예시:

```bash
C:\Users\<사용자이름>\.ssh\id_rsa
C:\Users\<사용자이름>\.ssh\id_rsa.pub

~/.ssh/id_rsa
~/.ssh/id_rsa.pub
```

생성되어 있지 않다면 아래와 같은 명령어를 실행하여 생성할 수 있습니다:

```bash
ssh-keygen -t rsa -b 4096 -C "your_email@example.com"
```

## Jupyter 작업 생성

사용자는 Jupyter 작업을 생성해야 합니다. 구체적인 생성 방법은 [대화형 작업](../quick-start/interactive.md)을 참조하세요. 예시 작업은 다음과 같습니다:

![](./img/vscode-ssh/job.webp)

"대화형 페이지"를 클릭하여 Jupyter Notebook으로 이동합니다.

![](./img/vscode-ssh/jupyter.webp)

## 컨테이너 내에서 SSHD 설치

**SSHD (SSH Daemon)**: SSHD는 SSH 서비스의 데몬으로 지정된 포트에서 SSH 연결 요청을 수신하고 해당 서비스를 제공합니다. SSHD를 설치하고 실행함으로써 컨테이너는 SSH 프로토콜을 통해 원격 접근이 가능해집니다.

### **OpenSSH 서버 설치**

```bash
sudo apt update
sudo apt install -y openssh-server
```

### SSH 서비스가 정상적으로 시작되었는지 확인

다음 명령어를 실행하여 `sshd`가 설치되어 있는지 수동으로 확인할 수 있습니다:

```bash
ps -ef | grep sshd
```

또는 `service` 명령어를 사용하여 SSHD의 실행 상태를 확인하고 관리할 수 있습니다:

**service 명령어를 사용하여 ssh 서비스 재시작**

```bash
sudo service ssh restart
```

**상태 확인**

```bash
sudo service ssh status
```

참고 출력 예시:

```bash
(base) liuxw24@jupyter-liuxw24-6838a-default0-0:~$ sudo service ssh restart
 * Restarting OpenBSD Secure Shell server sshd                       [ OK ]
(base) liuxw24@jupyter-liuxw24-6838a-default0-0:~$ sudo service ssh status
 * sshd is running
```

### 22 포트가 정상적으로 수신 중인지 확인

다음 명령어를 실행하여 `22` 포트가 수신 중인지 확인합니다:

```bash
sudo netstat -tuln | grep 22
```

모든 것이 정상이라면, `sshd` 서비스는 지정된 포트에서 연결을 수신합니다.

### SSH 무비밀 로그인 구성

`authorized_keys`를 사용하여 무비밀 로그인을 구성할 수 있습니다. 개인 컴퓨터에서 생성한 공개 키(`id_rsa.pub`)를 서버에 업로드합니다(이전 섹션에서 설명한 것처럼).

- `.ssh` 폴더가 없으면 다음 명령어로 생성하고 적절한 권한을 설정합니다:

```bash
mkdir ~/.ssh
chmod 700 ~/.ssh
```

- 로컬의 `id_rsa.pub` 파일 내용을 `~/.ssh/authorized_keys` 파일에 추가합니다:

```bash
# 로컬 id_rsa.pub 파일 내용을 ~/.ssh/authorized_keys에 복사
vim ~/.ssh/authorized_keys
# authorized_keys에 적절한 권한 설정
chmod 600 ~/.ssh/authorized_keys
```

## NodePort 규칙 설정

외부 액세스 규칙에서 **NodePort 규칙**을 설정하여 서비스 포트를 노출시킬 수 있습니다. NodePort 포트를 사용하여 VSCode에서 Jupyter 컨테이너에 연결할 수 있습니다.

NodePort 규칙은 사용자가 클러스터 노드의 IP 주소와 지정된 포트 번호를 통해 서비스에 액세스할 수 있도록 허용합니다. Kubernetes는 서비스에 포트 범위 30000에서 32767 사이에 포트를 할당하고, 외부에서 해당 포트 번호를 통해 연결할 수 있도록 합니다.

Jupyter 작업의 세부 정보 페이지에서 NodePort 규칙을 생성하십시오. 상세한 생성 절차는 [NodePort 액세스 규칙 설정](../toolbox/external-access/nodeport-rule.md)을 참조하십시오.

![](./img/vscode-ssh/nodeport.webp)

**필드 설명**:

- **컨테이너 포트 번호** (`containerPort`): **22** 포트를 선택하여 SSH 서비스에 사용합니다.
- **클러스터 노드 주소** (`address`): 클러스터의 임의 노드 IP 주소, 여기서는 `192.168.5.30`입니다.
- **할당된 NodePort 포트** (`nodePort`): Kubernetes는 서비스에 포트 범위 30000에서 32767 사이의 포트 번호를 자동으로 할당합니다. 여기서는 `32310`입니다.

## VSCode 설정

### Remote-SSH 확장 설치

VSCode에서 Remote-SSH 확장을 설치하려면 아래와 같이 수행합니다:

![](./img/vscode-ssh/remote-ssh.webp)

### **Remote.SSH Config** 파일 설정

설정에서 **Remote.SSH Config** 파일의 경로를 지정합니다:

![](./img/vscode-ssh/setting.webp)

config 파일의 예시 설정은 다음과 같습니다:

```yaml
Host 192.168.5.30
HostName 192.168.5.30
LogLevel verbose
IdentityFile C:\Users\lxw\.ssh\id_rsa
Port 32310
User liuxw24
```

각 필드 설명:

- `Host`: Host IP, NodePort 규칙의 `Host IP`를 참조합니다. 예시에서는 `192.168.5.30`입니다.
- `IdentityFile`: 개인 키 파일의 접근 경로를 지정합니다.
- `Port`: 연결할 포트 번호, NodePort 규칙의 `NodePort 포트 번호`를 참조합니다. 예시에서는 `32310`입니다.
- `User`: 사용자 이름입니다.

설정이 완료되면 VSCode를 통해 NodePort로 Jupyter 컨테이너에 성공적으로 연결할 수 있습니다:

![](./img/vscode-ssh/connected.webp)