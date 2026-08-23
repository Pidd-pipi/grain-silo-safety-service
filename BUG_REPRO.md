# BUG_REPRO: grain-silo-safety-service-008

## Bug 是什么
状态机流转历史无上限增长，配置项完全不生效：
- `OpsStateMachine.Move`（ops_state.go）`m.history = append(...)` 没有按配置收敛，历史切片只增不减；
- `config.MaxHistory()` 写死返回 500 且不读 `MAX_HISTORY` 环境变量，是无人调用的死代码（状态机构造没有接收容量参数）；
- `History()` 返回内部切片共享底层数组，调用方修改会污染存储。

长跑后状态机历史与内存持续增长。

## 如何触发
1. 长时间运行服务，持续进行工单状态流转（`POST /api/ops/records/{id}/transition`）。
2. 观察内存随历史增长；设置 `MAX_HISTORY` 环境变量后重启，历史仍不收敛。
3. 用大量流转（如 2000 次 active↔paused）后读取 `History()`，长度不封顶；改动返回的历史还会污染内部存储。

## 真实错误信息
无 panic；现象是历史长度无上限：
- 2000 次流转后 `len(History()) == 2000`（配置上限 500 未生效）；
- `MAX_HISTORY=100` 设置后 `MaxHistory()` 仍返回 500；
- 调用方修改 `History()` 返回的切片后，再次读取仍能看到被改内容。
