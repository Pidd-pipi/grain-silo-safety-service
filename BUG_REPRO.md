# BUG_REPRO: grain-silo-safety-service-010

## Bug 是什么
工单状态流转的「先记后落库」与错误映射错乱：
- `ops_service.go` 的 `Transition` 先调用 `state.Move`（写入状态机历史）再 `store.Update`（落库），版本冲突失败后历史里残留未发生的流转（幽灵流转），审计也可能在提交前发出；
- `ops_http.go` 的 `opsWriteError` 把 `transition`（非法流转）和 `conflict`（修订冲突）都映射成 500，掩盖真实错误类别。

## 如何触发
1. `POST /api/ops/records/wo-001/transition` 带错误版本号（如 `{"expected":99,"target":"paused"}`）→ 返回 500（应 409），且状态机历史里多了一条未发生的流转。
2. `POST /api/ops/records/wo-002/transition` 传非法目标（如 queued → paused）→ 返回 500（应 4xx）。
3. 成功流转后，响应里的 `revision` 是旧值（1）而不是落库后的新值（2）。

## 真实错误信息
```
{"error":"operations revision conflict"}  → 状态码 500（应 409）
{"error":"operations status transition is not allowed"} → 状态码 500（应 4xx）
```
失败后 `GET /api/ops/records/wo-001/audit` 能看到本不该出现的 `status_changed` 事件。
