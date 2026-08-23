# Grain Silo Safety Service

This standard-library HTTP service tracks grain-silo safety observations, formal inspection records with a review workflow, maintenance work orders, and threshold alerting for silo temperature and moisture.

## Layout

`config`, `domain`, `store`, `validation`, `health`, and `api` provide the service layers; the ops/inspection/alert HTTP groups live in the main package. The browser client is embedded from `web`.

## Run and test

Set `PORT` to choose a port; the default is `8080`.

```text
cd backend && go build ./...
cd backend && go test ./...
cd backend && PORT=8080 go run .
```

## Endpoints

- `GET /healthz` — health check.
- `GET /api/silos` — list silo safety observations.
- `POST /api/silos/{id}/inspect` — record a quick inspection finding (`{"finding":"..."}`). Empty findings return `400`; unknown silos `404`; clear silos `409`.
- `POST /api/inspections` — open a formal inspection record (`{"siloId":"...","finding":"..."}`).
- `GET /api/inspections` — list inspection records (`silo`, `status`, `severity`, `page`, `page_size`).
- `POST /api/inspections/{id}/review` / `POST /api/inspections/{id}/close` — advance an inspection record.
- `GET /api/ops/records` — list maintenance work orders (query: `subject`, `status`, `priority`, `owner`, `page`, `page_size`).
- `POST /api/ops/records` — create a work order (`{"subject":"...","owner":"...","priority":"high","labels":{"site":"west"}}`).
- `GET /api/ops/records/{id}` / `POST /api/ops/records/{id}/transition` / `GET /api/ops/records/{id}/audit` — work order detail, status transition, audit trail.
- `GET /api/ops/snapshot` — work order summary counts.
- `GET /api/alerts/rules` / `POST /api/alerts/rules` — alert threshold rules.
- `GET /api/alerts/events` — emitted alert events (a background sweeper evaluates rules periodically).

## Engineering Notes

请求保留请求标识并经过恢复与超时保护；状态写入使用版本校验，错误通过可识别的领域错误返回。巡检记录带并发容量保护，状态机保留最近流转历史，审计与告警事件持续累积供运营查看。
