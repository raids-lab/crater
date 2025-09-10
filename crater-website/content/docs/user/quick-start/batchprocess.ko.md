---
title: 사용자 정의 작업
description: PyTorch 손글씨 인식 GPU 훈련 작업 제출
---
# 배치 작업

## PyTorch 손글씨 인식 GPU 훈련 작업 제출

Crater의 **단일 머신 배치 작업**은 특정 환경에서 주어진 명령어에 따라 실행되고 결과를 얻는 데 사용됩니다.

## 코드 업로드

### 로컬에서 코드 생성

실행할 코드 파일을 로컬에서 생성합니다. 배치 시스템은 코드 디버깅 과정에서 많은 불편함을 줄 수 있으므로, 로컬에서 성공적으로 디버깅되고 정상적으로 실행되는 코드만 배치 시스템에 적용하는 것이 좋습니다. 이는 전체 업무 흐름의 원활한 진행과 안정성을 보장하기 위해서입니다.

```python
# 이 코드는 github PyToch 손글씨 인식 GPU 훈련 작업 예제를 참고하였습니다.
import argparse
import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
from torchvision import datasets, transforms
from torch.optim.lr_scheduler import StepLR


class Net(nn.Module):
    def __init__(self):
        super(Net, self).__init__()
        self.conv1 = nn.Conv2d(1, 32, 3, 1)
        self.conv2 = nn.Conv2d(32, 64, 3, 1)
        self.dropout1 = nn.Dropout(0.25)
        self.dropout2 = nn.Dropout(0.5)
        self.fc1 = nn.Linear(9216, 128)
        self.fc2 = nn.Linear(128, 10)

    def forward(self, x):
        x = self.conv1(x)
        x = F.relu(x)
        x = self.conv2(x)
        x = F.relu(x)
        x = F.max_pool2d(x, 2)
        x = self.dropout1(x)
        x = torch.flatten(x, 1)
        x = self.fc1(x)
        x = F.relu(x)
        x = self.dropout2(x)
        x = self.fc2(x)
        output = F.log_softmax(x, dim=1)
        return output


def train(args, model, device, train_loader, optimizer, epoch):
    model.train()
    for batch_idx, (data, target) in enumerate(train_loader):
        data, target = data.to(device), target.to(device)
        optimizer.zero_grad()
        output = model(data)
        loss = F.nll_loss(output, target)
        loss.backward()
        optimizer.step()
        if batch_idx % args.log_interval == 0:
            print('Train Epoch: {} [{}/{} ({:.0f}%)]\tLoss: {:.6f}'.format(
                epoch, batch_idx * len(data), len(train_loader.dataset),
                100. * batch_idx / len(train_loader), loss.item()))
            if args.dry_run:
                break


def test(model, device, test_loader):
    model.eval()
    test_loss = 0
    correct = 0
    with torch.no_grad():
        for data, target in test_loader:
            data, target = data.to(device), target.to(device)
            output = model(data)
            test_loss += F.nll_loss(output, target, reduction='sum').item()  # 배치 손실 누적
            pred = output.argmax(dim=1, keepdim=True)  # 최대 로그 확률의 인덱스
            correct += pred.eq(target.view_as(pred)).sum().item()

    test_loss /= len(test_loader.dataset)

    print('\nTest set: Average loss: {:.4f}, Accuracy: {}/{} ({:.0f}%)\n'.format(
        test_loss, correct, len(test_loader.dataset),
        100. * correct / len(test_loader.dataset)))


def main():
    # 훈련 설정
    parser = argparse.ArgumentParser(description='PyTorch MNIST Example')
    parser.add_argument('--batch-size', type=int, default=64, metavar='N',
                        help='훈련용 입력 배치 크기 (기본값: 64)')
    parser.add_argument('--test-batch-size', type=int, default=1000, metavar='N',
                        help='테스트용 입력 배치 크기 (기본값: 1000)')
    parser.add_argument('--epochs', type=int, default=2, metavar='N',
                        help='훈련할 에포크 수 (기본값: 2)')
    parser.add_argument('--lr', type=float, default=1.0, metavar='LR',
                        help='학습률 (기본값: 1.0)')
    parser.add_argument('--gamma', type=float, default=0.7, metavar='M',
                        help='학습률 스탭 감마 (기본값: 0.7)')
    parser.add_argument('--no-cuda', action='store_true', default=False,
                        help='CUDA 훈련 비활성화')
    parser.add_argument('--no-mps', action='store_true', default=False,
                        help='macOS GPU 훈련 비활성화')
    parser.add_argument('--dry-run', action='store_true', default=False,
                        help='단일 실행 빠르게 확인')
    parser.add_argument('--seed', type=int, default=1, metavar='S',
                        help='난수 시드 (기본값: 1)')
    parser.add_argument('--log-interval', type=int, default=10, metavar='N',
                        help='훈련 상태 로깅 전 대기 배치 수')
    parser.add_argument('--save-model', action='store_true', default=True,
                        help='현재 모델 저장')
    args = parser.parse_args()
    use_cuda = not args.no_cuda and torch.cuda.is_available()
    use_mps = not args.no_mps and torch.backends.mps.is_available()

    torch.manual_seed(args.seed)

    if use_cuda:
        device = torch.device("cuda")
    elif use_mps:
        device = torch.device("mps")
    else:
        device = torch.device("cpu")

    train_kwargs = {'batch_size': args.batch_size}
    test_kwargs = {'batch_size': args.test_batch_size}
    if use_cuda:
        cuda_kwargs = {'num_workers': 1,
                       'pin_memory': True,
                       'shuffle': True}
        train_kwargs.update(cuda_kwargs)
        test_kwargs.update(cuda_kwargs)

    transform = transforms.Compose([
        transforms.ToTensor(),
        transforms.Normalize((0.1307,), (0.3081,))
    ])
    dataset1 = datasets.MNIST('../data', train=True, download=True,
                              transform=transform)
    dataset2 = datasets.MNIST('../data', train=False,
                              transform=transform)
    train_loader = torch.utils.data.DataLoader(dataset1, **train_kwargs)
    test_loader = torch.utils.data.DataLoader(dataset2, **test_kwargs)

    model = Net().to(device)
    optimizer = optim.Adadelta(model.parameters(), lr=args.lr)

    scheduler = StepLR(optimizer, step_size=1, gamma=args.gamma)
    for epoch in range(1, args.epochs + 1):
        train(args, model, device, train_loader, optimizer, epoch)
        test(model, device, test_loader)
        scheduler.step()

    if args.save_model:
        torch.save(model.state_dict(), "mnist_cnn.pt")


if __name__ == '__main__':
    main()
```

### 로컬 코드 업로드

시스템에서 데이터 관리 파일 시스템을 열고 사용자 공간으로 이동합니다.

![openuserspace](./assets/openuserspace.webp)

프로젝트에 대해 별도로 폴더를 생성하여 관련 파일을 가져올 수 있습니다.

![createfile](./assets/createfile.webp)

이 예제에서는 train.py 파일을 mnist 폴더에 업로드합니다.

![uploadfile](./assets/uploadfile.webp)

## 작업 제출

배치 작업에서 사용자 정의 단일 머신 배치 작업을 선택합니다.

![selectbatchprocess](./assets/selectbatchprocess.webp)

### 작업 정보 입력

![filljobinfo](./assets/filljobinfo.webp)

### 컨테이너 이미지 선택

공용 이미지나 사용자 정의 이미지를 선택할 수 있습니다. 공용 이미지에 대한 설명과 사용자 정의 이미지를 생성하는 방법은 관련 문서에서 확인할 수 있습니다.

### 시작 명령

#### 예시 명령

```bash
cd /mnt/mnist;
python train.py
```

1. /mnt/mnist 폴더 열기
2. python을 사용하여 train.py 실행

#### 주의사항

- 명령 사이에 `;`를 사용하여 분리해야 하며, 이는 동일한 shell 세션에서 순차적으로 실행됩니다.
- 명령에 특수 문자나 공백이 포함된 경우 전체 명령 문자열을 따옴표로 감싸는 것이 좋습니다. 이는 해석 오류를 방지하기 위해서입니다.

이 방법을 통해 컨테이너를 시작할 때 이미지의 기본 시작 명령을 덮어쓸 수 있으며, 여러 줄의 명령을 실행할 수 있습니다.

### 데이터 마운트

초기 상태는 전체 사용자 파일 시스템이 마운트되어 있습니다. 일반적으로 파일 시스템과 데이터셋은 모두 /mnt/ 폴더에 마운트됩니다.

![mount](./assets/mount.webp)

예를 들어, 사용자 공간의 mnist 폴더를 가져오는 경우 cd /mnt/mnist; 명령을 통해 이 폴더에 접근할 수 있습니다.

![filemount](./assets/filemount.webp)

폴더와 마찬가지로, 데이터셋 마운트는 데이터셋이 포함된 전체 폴더를 /mnt/ 폴더에 마운트하는 것입니다. 아래와 같습니다.

![datasetmount](./assets/datasetmount.webp)

![opendataset](./assets/opendataset.webp)

## 작업 실행 상태 확인

작업을 성공적으로 생성한 후, 배치 작업 목록에서 방금 생성한 작업을 볼 수 있습니다. 작업 이름을 클릭하여 세부 정보를 확인할 수 있습니다.

![openjob](./assets/openjob.webp)

![checkjob](./assets/checkjob.webp)

작업 실행 중에는 파일을 열 수 없는 경우나 프로그램 실행 중 문제가 발생할 수 있습니다. 작업 실행 상태를 확인하려면 로그를 확인할 수 있습니다.

![checklog](./assets/checklog.webp)

이 시점에서는 프로그램이 데이터셋을 다운로드하고 있으며, 시스템 대리 문제로 인해 다운로드 속도가 느릴 수 있습니다. 데이터셋을 미리 다운로드하여 업로드하는 것이 좋습니다.

![checkjobincmd](./assets/checkjobincmd.webp)

상태가 **완료됨**이 되면 배치 작업이 성공적으로 실행되었음을 나타내며, 실패하면 문제가 발생했음을 의미합니다. 로그를 통해 확인할 수 있습니다.

![jobstatus](./assets/jobstatus.webp)

#### 작업 실행 상태 확인 방법

새로운 배치 작업을 생성할 때 환경 문제를 마주치는 경우, **시작 명령**에 sleep 명령을 추가하여 디버깅할 수 있습니다. 예를 들어:

```
sleep 600;//프로그램을 10분간 일시 중지
```

그런 다음 터미널 및 로그를 통해 원하는 정보를 확인할 수 있습니다.

![openterminal](./assets/openterminal.webp)

## 작업 결과 저장

이미지 공간의 현재 폴더에 저장된 내용은 사용자 공간의 원본 폴더에 저장됩니다. 예시 코드에서 실행 후 mnist_cnn.pt 파일이 train.py가 있는 원본 폴더 mnist에 저장됩니다.

```
torch.save(model.state_dict(), "mnist_cnn.pt")
```

![jobsave](./assets/jobsave.webp)