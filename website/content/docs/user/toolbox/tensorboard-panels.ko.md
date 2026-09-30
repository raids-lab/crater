---
title: TensorBoard 패널
description: TensorBoard 로그 패널 생성, 접근 및 관리
---

# TensorBoard 패널

Crater에서는 개인 디렉터리 또는 기존 작업의 로그로 TensorBoard 패널을 만들 수 있습니다. 개인 패널 페이지나 작업 메뉴에서 시작합니다.

## 로그 기록

단일 노드, Jupyter, WebIDE, PyTorch DDP 및 TensorFlow PS 작업 양식에서 `TENSORBOARD_LOGDIR`을 설정할 수 있습니다. 기본값은 작업 생성 후 백엔드에서 `/home/<user>/tensorboard-runs/<unique-job-name>`으로 해석되므로 표시 이름이 같은 작업도 로그 디렉터리를 공유하지 않습니다. 사용자 지정 디렉터리는 절대 경로여야 합니다.

TensorFlow PS 템플릿은 여러 worker가 같은 이벤트를 중복 기록하지 않도록 기본적으로 worker-0(chief)이 summary를 기록합니다.

## 패널 생성

1. **TensorBoard** 생성 페이지를 엽니다.
2. 개인 디렉터리를 추가하거나 현재 사용자가 소유한 작업을 선택합니다. 작업 선택기는 모든 결과 페이지를 불러옵니다.
3. 0~10개의 소스를 추가할 수 있습니다. 여러 작업 소스는 `/tensorboard-runs/<job-name>` 아래에 각각 마운트되고 run은 재귀적으로 검색됩니다.
4. 패널을 제출합니다. 스케줄링 대기열에 들어가며 Pod 시작 후 최대 4일 동안 실행됩니다. 사용자마다 대기 중이거나 활성 상태인 패널을 최대 10개 유지할 수 있습니다.

개인 디렉터리는 현재 사용자의 `/home/<user>` 아래에 있어야 합니다. 작업 소스는 자신의 작업과 해당 작업에 허용된 스토리지 마운트만 참조할 수 있습니다. 생성 페이지는 디렉터리 내용을 검사하지 않으므로 event 파일이 있는지 확인하십시오.

## 상태, 접근 및 정리

상태는 `pending`, `starting`, `ready`, `failed`, `expired`입니다. Volcano Job phase와 Pod Ready 조건으로 판단하며 `pending`은 일반 VCJob의 Pending phase와 일치합니다. Service, Ingress 또는 EndpointSlice를 별도로 검사하지 않습니다.

패널을 열 때 Crater는 로그인 사용자, 패널 존재 여부 및 소유권을 확인한 후 해당 패널 경로 전용 단기 접근 세션을 발급합니다. URL만 알아서는 검사를 우회할 수 없습니다. 삭제 또는 자동 정리 후 리소스가 제거되면 URL도 무효화됩니다.

## 문제 해결

- `pending` 상태가 계속됨: 스케줄링 대기열에서 호환 CPU/메모리를 기다리고 있습니다. 관리자에게 대기열과 노드 용량 확인을 요청하십시오.
- `starting` 상태가 계속됨: 관리자에게 이미지 pull과 Pod 이벤트 확인을 요청하십시오.
- `failed`: 표시된 안전한 영어 요약을 확인하십시오. 클러스터 내부 정보는 반환되지 않습니다.
- 차트가 없음: 선택한 경로의 event 파일과 학습 프로그램 출력 경로를 확인하십시오.
