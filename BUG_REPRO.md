# BUG_REPRO: grain-silo-safety-service-009

## Bug 是什么
HTTP 层错误被静默吞掉，多处失败被伪装成成功：
- `runtime.go` 的 `recoveryMiddleware` 捕获 panic 后只打日志、不写 500，接口 panic 时返回 200 空内容；
- `api/router.go` 静态资源处理器在文件缺失时不返回 404，而是回吐 `index.html` 并返回 200；
- `health/handler.go` 的 `Handler` 无视就绪检查（ready 报错也恒返回 200 + `{"status":"ok"}`），健康检查永远报健康。

## 如何触发
1. 让任一处理器 panic（如异常输入触发内部错误）→ 响应 200 空 body（应为 500）。
2. 请求不存在的静态资源 `GET /no-such.js` → 返回 200 首页内容（应为 404）。
3. 传入就绪失败（store 不可用）时请求 `GET /healthz` → 仍返回 200 `{"status":"ok"}`（应为非 200 且不报 ok）。

## 真实错误信息
无 panic 输出；现象是错误被吞：
```
panic 请求 → 200 空内容（应 500）
GET /no-such.js → 200 index.html（应 404）
GET /healthz（依赖故障）→ 200 {"status":"ok"}（应非 200）
```
