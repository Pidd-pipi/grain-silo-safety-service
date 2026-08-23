# BUG_REPRO: grain-silo-safety-service-003

## Bug 是什么
错误链多处断裂导致公开接口状态码错乱：
- `domain.RecordInspection` 用 `%v` 包装 `ErrSiloRejected`（应为 `%w`），`errors.Is` 失效；
- `api/inspectSilo` 对「未知筒仓」用字符串比较替代 `errors.Is(err, ErrSiloNotFound)`；
- `ops_errors.go` 的 `wrapOps` 丢弃 `Cause` 且 `OpsError.Unwrap` 缺失，`errors.Is/As` 全部失效。

连锁结果：巡检 clear 筒仓、巡检未知筒仓、查询不存在的工单、重复创建工单，全部被误判成 500（应为 409/404）。

## 如何触发
1. `POST /api/silos/silo-02/inspect`（clear 筒仓）→ 期望 409，实际 500。
2. `POST /api/silos/nope/inspect`（未知筒仓）→ 期望 404，实际 500。
3. `GET /api/ops/records/nope`（不存在工单）→ 期望 404，实际 500。
4. `POST /api/ops/records` 重复创建 wo-001 → 期望 409，实际 500。

## 真实错误信息
```
{"error":"internal"} 或 500 状态码
```
`errors.Is(err, domain.ErrSiloRejected)`、`errors.Is(err, ErrOpsNotFound)` 等在 bug 环境下全部返回 false（错误链断链）。
