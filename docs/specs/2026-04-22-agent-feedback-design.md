# Agent Feedback / Rating System

## Overview

为 AI Chat 的每条 assistant message 和每个 turn 提供用户反馈机制（thumbs up/down + 可选详情），
用于追踪具体案例质量并支持人工评分实验数据收集。

## Data Model

### `agent_feedbacks` 表（新建）

| Column        | Type           | Constraint / Default             | Description                                                |
|---------------|----------------|----------------------------------|------------------------------------------------------------|
| id            | uint PK        | auto-increment                   | 主键                                                       |
| session_id    | uuid           | NOT NULL, INDEX                  | 所属会话                                                   |
| user_id       | uint           | NOT NULL, INDEX                  | 评价人                                                     |
| account_id    | uint           | NOT NULL, INDEX                  | 租户隔离                                                   |
| target_type   | varchar(16)    | NOT NULL, INDEX                  | `message` 或 `turn`                                        |
| target_id     | varchar(128)   | NOT NULL, INDEX                  | message → message.id (uint), turn → turn_id (uuid)         |
| rating        | smallint       | NOT NULL                         | `1` = thumbs up, `-1` = thumbs down                       |
| tags          | jsonb          | nullable                         | 预定义标签数组 `["inaccurate","irrelevant","wrong_direction"]` |
| dimensions    | jsonb          | nullable                         | 多维度评分 `{"relevance":4,"accuracy":2,"usefulness":3}`     |
| comment       | text           | nullable                         | 自由文本                                                   |
| status        | varchar(16)    | NOT NULL, DEFAULT 'draft'        | `draft` / `submitted`                                      |
| submitted_at  | timestamptz    | nullable                         | 提交时间戳                                                 |
| created_at    | timestamptz    |                                  |                                                            |
| updated_at    | timestamptz    |                                  |                                                            |

**唯一约束**: `(user_id, target_type, target_id)` — 每个用户对同一目标只能有一条反馈。

**不变性规则**: `status=submitted` 后拒绝一切修改（后端返回 409 Conflict）。

## Predefined Tags

| Key               | Label (zh-CN)    |
|-------------------|------------------|
| inaccurate        | 不准确           |
| irrelevant        | 不相关           |
| incomplete        | 不完整           |
| wrong_direction   | 思路方向错误     |
| too_slow          | 响应太慢         |
| helpful           | 很有帮助         |
| clear             | 表述清晰         |

## Dimension Scores

维度评分范围 1-5，前端可选展开：

- relevance (相关性)
- accuracy (准确性)
- usefulness (有用性)

## API Endpoints

所有路由挂在 `/api/v1/agent/feedbacks` 下，属于 Protected（登录用户）。

### 1. `PUT /api/v1/agent/feedbacks`

创建或更新反馈（upsert by user_id + target_type + target_id）。

**Request Body:**
```json
{
  "sessionId": "uuid",
  "targetType": "message",
  "targetId": "123",
  "rating": 1,
  "tags": ["helpful", "clear"],
  "dimensions": {"relevance": 5, "accuracy": 4, "usefulness": 5},
  "comment": "optional text"
}
```

**Rules:**
- 如果记录不存在 → 创建 draft
- 如果 status=draft → 更新所有字段
- 如果 status=submitted → 返回 409 Conflict

### 2. `POST /api/v1/agent/feedbacks/submit`

将 draft 反馈标记为 submitted（不可逆）。

**Request Body:**
```json
{
  "sessionId": "uuid",
  "targetType": "message",
  "targetId": "123"
}
```

**Rules:**
- 只允许 draft → submitted
- 同时写入 `submitted_at`
- 已提交的返回 409

### 3. `GET /api/v1/agent/feedbacks?sessionId=xxx`

获取当前用户在某个 session 下的所有反馈。

**Response:**
```json
{
  "code": 200,
  "data": [
    {
      "id": 1,
      "sessionId": "uuid",
      "targetType": "message",
      "targetId": "123",
      "rating": -1,
      "tags": ["inaccurate"],
      "dimensions": {"relevance": 2, "accuracy": 1, "usefulness": 2},
      "comment": "答案完全不对",
      "status": "submitted",
      "submittedAt": "2026-04-22T10:00:00Z",
      "createdAt": "2026-04-22T09:55:00Z",
      "updatedAt": "2026-04-22T10:00:00Z"
    }
  ]
}
```

### 4. `GET /api/v1/agent/feedbacks/stats` (Admin)

管理员统计接口，按时间范围聚合反馈数据。

**Query Params:** `from`, `to` (ISO timestamps, optional)

**Response:**
```json
{
  "code": 200,
  "data": {
    "total": 100,
    "thumbsUp": 72,
    "thumbsDown": 28,
    "avgDimensions": {"relevance": 3.8, "accuracy": 3.5, "usefulness": 3.9},
    "topTags": [
      {"tag": "helpful", "count": 45},
      {"tag": "inaccurate", "count": 15}
    ]
  }
}
```

## Frontend Design

### Message-level Feedback

在每条 `kind === 'message'` (assistant) 的消息气泡底部，增加 thumbs up/down 按钮。

**交互流程：**
1. 默认显示 `👍 👎` 两个 icon button（ghost style，低干扰）
2. 用户点击任一按钮 → 立即调用 PUT upsert（draft）→ 按钮变为选中态
3. 同时展开可折叠详情区域（Collapsible），包含：
   - 预定义标签 chip 多选
   - 三个维度滑块（relevance / accuracy / usefulness，1-5）
   - 自由文本 textarea
   - 「提交反馈」按钮 → 调用 submit 接口
4. 提交后所有控件变为只读，显示「已提交」标记
5. 未提交前可以反复修改 rating / tags / dimensions / comment

### Turn-level Feedback

在 timeline 模式下，每个 turn 区块末尾显示同样的反馈组件（targetType = 'turn'）。

### Session Restore

加载历史会话时，通过 `GET /feedbacks?sessionId=xxx` 获取已有反馈，回填到对应消息/turn 上。

## Migration

在 `migrate.go` 中新增 migration ID `"202604220001"`，使用 `tx.AutoMigrate(&model.AgentFeedback{})` 创建表。
