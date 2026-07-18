---
title: Model and Dataset Download Quotas
description: Understand download quotas, public-resource reuse, pause, retry, and resume accounting
---

## Operations that consume quota

Administrators can configure two per-user limits: the number of tasks that are pending or downloading at the same time, and the number of successful downloads in a rolling time window. Creating a Kubernetes download Job reserves a rolling-window slot immediately so users cannot submit an unbounded number of tasks while earlier downloads are still running.

Creating a new download, retrying a failed task, and resuming a paused task all check the quota of the user performing the action. A successful task starts consuming the rolling window from its actual completion time. A failed, paused, deleted, or unsuccessfully submitted Job releases its reservation and does not count as a successful download.

## Reusing public resources

Crater stores one public copy of each model or dataset. Reusing a completed public resource, or joining a public task that is already pending, downloading, or paused, creates no new Job and consumes no download quota. Crater still records that the current user explicitly requested the resource.

The requester count only records demand. It does not change access or sharing permissions; files in public storage keep the platform's existing public-directory access rules.

## Pause, retry, and resume

Pausing saves the latest logs, deletes the running Kubernetes Job, and stops transfer. Files already written remain in place. Resuming creates a new Job, attempts to continue the partial download, and checks the quota of the user who resumes it.

Retrying a failed task is also charged to the user who performs the retry. Released attempts from other users do not consume the current user's quota. When quotas are disabled or a user is whitelisted, Crater does not block the Job, but it still records the lifecycle so already-running tasks are counted correctly if quotas are enabled later or the user is removed from the whitelist.
