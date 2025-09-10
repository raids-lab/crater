---
title: SSH 기능을 사용하여 빠르게 연결

description: 사용자가 컨테이너에 더 편리하게 연결할 수 있도록 이 플랫폼은 SSH 기능을 제공합니다. SSH 비밀번호 없이 로그인을 설정한 후에는 사용자가 터미널이나 VSCode를 통해 컨테이너에 연결할 수 있는 명령어를 한 번 클릭으로 복사할 수 있습니다.
---

## SSH 비밀번호 없는 로그인 설정

`authorized_keys`를 사용하여 비밀번호 없는 로그인을 설정할 수 있으며, 일반적으로 `id_rsa.pub`인 공개 키를 서버에 업로드하면 됩니다 (공개 키 및 개인 키 생성 과정은 [VSCode로 Jupyter 컨테이너에 연결하기](./vscode-ssh.md)의 "로컬 컴퓨터에 공개 키 및 개인 키 생성 확인" 섹션을 참조하십시오).

- `.ssh` 폴더가 없으면 다음 명령어로 `.ssh` 폴더를 생성하고 적절한 권한을 설정할 수 있습니다:

```bash
mkdir ~/.ssh
chmod 700 ~/.ssh
```

- 로컬 컴퓨터의 공개 키를 `~/.ssh/authorized_keys` 파일에 추가합니다.

```bash
# 로컬 컴퓨터의 id_rsa.pub 파일 내용을 ~/.ssh/authorized_keys에 복사합니다
vim ~/.ssh/authorized_keys
# authorized_keys에 적절한 권한을 설정합니다
chmod 600 ~/.ssh/authorized_keys
```

이 과정은 **최초 연결 시에만 필요**하며, 설정이 완료되면 다시 연결할 때는 추가 작업이 필요하지 않습니다.

## SSHD가 포함된 이미지로 작업 생성

현재 플랫폼에서 제공하는 공식 이미지는 **SSHD가 포함**되어 있으므로, 추가 설치가 필요 없습니다 🚀.

자체 이미지를 사용하려면 생성한 이미지에 SSHD가 포함되어 있어야 합니다.

## 연결 명령어 한 번 클릭으로 복사

**작업 상세 페이지**로 이동한 후, 페이지 우상단의 **"SSH 연결"** 버튼을 클릭합니다.

![](./img/ssh-func/ssh-detail.webp)

클릭하면 아래와 같은 대화상자가 나타납니다:

![](./img/ssh-func/ssh-func.webp)

필요에 따라 터미널과 VSCode의 연결 명령어를 복사할 수 있습니다.

## 터미널 연결

터미널에서 복사한 명령어를 입력하면 컨테이너에 연결할 수 있습니다.

![](./img/ssh-func/terminal.webp)

## VSCode 연결

(1) VSCode에서 Remote-SSH 확장을 설치해야 합니다. 다음을 참조하십시오:

![](./img/ssh-func/remote-ssh.webp)

(2) VSCode 하단 좌측의 원격 연결 아이콘을 클릭하고, 나타나는 메뉴에서 **"Remote - SSH: Host에 연결"**을 선택합니다.

(3) 처음으로 연결하는 경우, VSCode는 운영 체제 유형을 선택하라는 메시지를 표시합니다. 해당 운영 체제(예: Linux)를 선택합니다.

(4) VSCode가 원격 서버에 필요한 구성 요소를 설치하는 데 시간이 걸릴 수 있습니다.

![](./img/ssh-func/download-server.webp)

(5) 설치가 완료되면, VSCode는 컨테이너에 연결할 수 있습니다.

![](./img/ssh-func/connect.webp)