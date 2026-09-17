# Project progress

This file is the roadmap index and the source of truth for task status. Detailed notes and acceptance evidence live in one file per task under [`docs/progress`](docs/progress).

## Validation workflow

Every task follows this state machine:

`pending` → `in_progress` → `awaiting_validation` → `completed`

- Implementation and automated verification may move a task to `awaiting_validation`.
- Only explicit user approval may move a task to `completed`.
- A task is committed only after approval, unless the user requests a different commit policy.
- Work on the next task does not start before the current task is approved.
- Manual checks are recorded as evidence; they are never silently treated as passed.

## Tasks

| ID | Phase | Task | Status | Progress file |
|---|---:|---|---|---|
| 00 | Setup | Repository and progress tracking | `completed` | [task-00-project-setup.md](docs/progress/task-00-project-setup.md) |
| 01 | 0 | Minimal reverse proxy spike | `completed` | [task-01-reverse-proxy-spike.md](docs/progress/task-01-reverse-proxy-spike.md) |
| 02 | 1 | YAML configuration and route matching | `completed` | [task-02-config-and-matching.md](docs/progress/task-02-config-and-matching.md) |
| 03 | 1 | Fault interface and first four faults | `completed` | [task-03-core-faults.md](docs/progress/task-03-core-faults.md) |
| 04 | 1 | Proxy pipeline and CORS | `completed` | [task-04-pipeline-and-cors.md](docs/progress/task-04-pipeline-and-cors.md) |
| 05 | 1 | Configuration hot reload | `completed` | [task-05-hot-reload.md](docs/progress/task-05-hot-reload.md) |
| 06 | 2 | Event bus and control plane | `completed` | [task-06-events-and-control-plane.md](docs/progress/task-06-events-and-control-plane.md) |
| 07 | 2 | Embedded web UI | `completed` | [task-07-web-ui.md](docs/progress/task-07-web-ui.md) |
| 08 | 2 | TCP reset and bandwidth faults | `completed` | [task-08-reset-and-bandwidth.md](docs/progress/task-08-reset-and-bandwidth.md) |
| 09 | 3 | Determinism and header override | `completed` | [task-09-determinism-and-override.md](docs/progress/task-09-determinism-and-override.md) |
| 10 | 3 | Scenario mode | `completed` | [task-10-scenario-mode.md](docs/progress/task-10-scenario-mode.md) |
| 11 | 3 | JSON payload mutation | `completed` | [task-11-payload-mutation.md](docs/progress/task-11-payload-mutation.md) |
| 12 | 3 | CI integrations and reports | `completed` | [task-12-ci-integrations.md](docs/progress/task-12-ci-integrations.md) |
| 13 | 4 | Packaging and distribution | `completed` | [task-13-packaging.md](docs/progress/task-13-packaging.md) |
| 14 | 4 | Final README | `completed` | [task-14-final-readme.md](docs/progress/task-14-final-readme.md) |
| 15 | Launch | Demonstration and launch preparation | `completed` | [task-15-launch.md](docs/progress/task-15-launch.md) |
| 16 | Release | Release and publication | `in_progress` | [task-16-release.md](docs/progress/task-16-release.md) |
| R1 | Review | Code quality and open-source readiness | `completed` | [task-R1-open-source-review.md](docs/progress/task-R1-open-source-review.md) |

## Phase acceptance checks

Phase-level checks are recorded in the final task of each phase. Checks involving a real frontend, a clean installation, publication, or a third-party service require manual approval and evidence.

Nothing is deployed or published before every other task is completed. Everything that pushes, publishes, or touches a third-party service is grouped in task 16.
