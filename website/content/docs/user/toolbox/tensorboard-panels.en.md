---
title: TensorBoard Panels
description: Create, access, and manage TensorBoard log panels
---

# TensorBoard Panels

Crater can create a TensorBoard panel from a personal directory or logs produced by existing jobs. Start from the personal panel page or a job action menu.

## Writing logs

Single-node, Jupyter, WebIDE, PyTorch DDP, and TensorFlow PS job forms can set `TENSORBOARD_LOGDIR`. After job creation, the backend resolves the default to `/home/<user>/tensorboard-runs/<unique-job-name>`. Jobs with the same display name therefore do not share a log directory. A custom directory must be absolute.

The TensorFlow PS template lets worker-0 (the chief) write summaries by default so workers do not duplicate the same event stream.

## Creating a panel

1. Open the **TensorBoard** creation page.
2. Add a personal directory or select jobs owned by the current user. The job selector loads all result pages.
3. Add zero to ten sources. Multiple job sources are mounted under `/tensorboard-runs/<job-name>`, where TensorBoard discovers runs recursively.
4. Submit the panel. It joins your scheduling queue and can run for up to four days after its Pod starts. Each user can keep at most ten pending or active panels.

A personal directory must be inside the current user's `/home/<user>` tree. A job source may reference only the user's own job and storage mounts that job is allowed to use. The creation page does not inspect directory contents, so verify that each path exists and contains TensorBoard event files.

## Status, access, and cleanup

Panel status is `pending`, `starting`, `ready`, `failed`, or `expired`. It is derived from the Volcano Job phase and Pod readiness; `pending` matches the Pending phase used by regular VCJobs. It is not an independent health check of the Service, Ingress, or EndpointSlice.

When a panel is opened, Crater verifies the signed-in user, current panel existence, and ownership before issuing a short-lived access session scoped to that panel path. Knowing the URL alone does not bypass this check. Deleting a panel or automatic cleanup removes its resources and invalidates the URL.

## Troubleshooting

- Stuck at `pending`: the panel is waiting for compatible CPU and memory in your scheduling queue; ask an administrator to inspect queue and node capacity.
- Stuck at `starting`: ask an administrator to inspect image pulls and Pod events.
- `failed`: read the panel error. Only a safe English summary is returned; cluster internals are not exposed.
- The page opens without charts: verify the selected path contains event files and matches the training process output directory.
