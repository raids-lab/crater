---
title: NodePort 접근 규칙
description: NodePort 규칙은 서비스 포트를 노출시키고, 외부 사용자가 클러스터 노드의 IP 주소와 지정된 포트 번호를 통해 접근할 수 있도록 허용합니다.
---

## 2.1 기능 소개

**NodePort 규칙**은 외부 IP를 통해 Kubernetes 클러스터 내 서비스에 직접 접근할 수 있도록 합니다. Ingress 규칙과는 달리, NodePort 규칙은 서비스의 포트를 노출시키고, 외부 사용자가 클러스터 노드의 IP 주소와 지정된 포트 번호를 통해 접근할 수 있도록 합니다. NodePort 규칙은 외부 접근이 필요한 애플리케이션, 예를 들어 **SSH 연결**에 적합합니다.

NodePort 규칙에서는 **Kubernetes가 30000에서 32767 포트 범위 내에서 서비스에 포트 번호를 자동으로 할당**합니다. 예를 들어, 클러스터 내 노드에 SSH 연결을 원할 경우, Kubernetes는 해당 서비스에 포트를 할당하고, 외부에서 이 포트 번호를 통해 연결할 수 있습니다.

**장점**:

- 외부 직접 접근이 필요한 애플리케이션, 예를 들어 SSH 연결에 적합합니다.
- 서비스에 포트를 자동으로 할당하여 설정을 간단하게 합니다.
- HTTP/HTTPS 프로토콜에 의존하지 않습니다.

**사용 시나리오**:

- 클러스터 내 노드에 SSH(포트 22)로 연결
- 특정 포트를 통해 접근해야 하는 다른 애플리케이션

**전달 경로**: `{nodeIP}:{nodePort}`를 통해 접근할 수 있습니다. 여기서 `nodeIP`는 클러스터의 임의의 노드 주소이고, `nodePort`는 할당된 포트 번호입니다.

![nodeport-intro](./img/nodeport-intro.webp)

설정이 완료되면 해당 Pod의 `Annotations`에 다음 내용을 확인할 수 있습니다. 이때 `nodeport.crater.raids.io`를 `key`로 사용합니다:

```yaml
metadata:
  annotations:
    crater.raids.io/task-name: tensorboard-example
    nodeport.crater.raids.io/smtp: '{"name":"smtp","containerPort":25,"address":"192.168.5.82","nodePort":30631}'
    nodeport.crater.raids.io/ssh: '{"name":"ssh","containerPort":22,"address":"192.168.5.82","nodePort":32513}'
    nodeport.crater.raids.io/telnet: '{"name":"telnet","containerPort":23,"address":"192.168.5.82","nodePort":32226}'
```

## 2.2 사용 예시

외부 IP 주소를 통해 특정 애플리케이션(예: SSH 연결)에 접근하려면 **NodePort 규칙**을 사용할 수 있습니다. 예를 들어, VSCode와 같은 도구를 통해 원격 개발을 위해 SSH 포트(22 포트)를 노출시키기 위해 NodePort 규칙을 구성할 수 있습니다.

**NodePort 외부 접근 규칙을 설정하는 단계는 다음과 같습니다:**

1. 작업 세부 정보 페이지에서 **"외부 접근 규칙 설정"**을 클릭합니다.

   ![ingress-entrance](./img/ingress-entrance.webp)

2. 팝업된 대화상자에서 **"NodePort 규칙 추가"**를 클릭하고, 해당 **규칙 이름**(소문자만 허용, 20자 이하, 중복 불가)과 **컨테이너 포트**를 입력한 후 저장을 클릭합니다.

   ![nodeport-new](./img/nodeport-new.webp)

3. 저장이 완료되면 **대응하는 NodePort 규칙**을 확인할 수 있습니다.

   ![nodeport-ssh](./img/nodeport-ssh.webp)

**예시 설정**:

```json
{
  "name": "ssh",
  "containerPort": 22,
  "address": "192.168.5.82",
  "nodePort": 32513
}
```

**필드 설명**:

- **컨테이너 포트 번호** (`containerPort`): 일반적으로 SSH 서비스에 사용되는 **22** 포트를 선택합니다.
- **클러스터 노드 주소** (`address`): 클러스터의 임의의 노드 IP 주소.
- **할당된 NodePort 포트** (`nodePort`): Kubernetes가 30000에서 32767 포트 범위 내에서 서비스에 포트 번호를 자동으로 할당합니다.

**접근 방법**:

- Kubernetes는 SSH 서비스에 포트 번호를 자동으로 할당하며, 외부 IP를 통해 이 포트를 사용하여 SSH 연결이 가능합니다(예: VSCode를 통한 원격 개발).
- 예를 들어, Kubernetes는 해당 서비스에 포트(예: `32513`)를 할당하고, `ssh user@<node-ip>:32513`을 통해 연결할 수 있습니다.

VSCode에서 NodePort를 통해 원격 Jupyter Notebook에 연결하는 효과는 다음과 같습니다:

![vscode-nodeport-ssh](./img/vscode-nodeport-ssh.webp)