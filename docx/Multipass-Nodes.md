# Multipass Ubuntu 节点信息

核验日期：2026-07-27（Asia/Shanghai）

## 节点概览

| 节点 | 状态 | IPv4 | 系统 | 架构 | CPU | 内存 | 磁盘 | cloud-init |
| --- | --- | --- | --- | --- | ---: | ---: | ---: | --- |
| `node1` | Running | `192.168.252.2` | Ubuntu 24.04.4 LTS | `aarch64` | 2 | 4G（实例内显示 3.8 GiB） | 20G（实例内显示 19.3 GiB） | done |
| `node2` | Running | `192.168.252.3` | Ubuntu 24.04.4 LTS | `aarch64` | 2 | 4G（实例内显示 3.8 GiB） | 20G（实例内显示 19.3 GiB） | done |
| `node3` | Running | `192.168.252.4` | Ubuntu 24.04.4 LTS | `aarch64` | 2 | 4G（实例内显示 3.8 GiB） | 20G（实例内显示 19.3 GiB） | done |

镜像 SHA-256：`7df0201546f75b8bcc1044594c806c35749421ad3c9bc1be2a3ab806cfae39cc`

## 登录信息

- 用户名：`ubuntu`
- 密码：`123456`，账户密码状态为 `P`（已设置）
- 认证方式：Multipass 注入并管理的 SSH 密钥
- SSH 密码认证：`PasswordAuthentication no`，当前不允许直接使用密码登录 SSH

进入各实例：

```bash
multipass shell node1
multipass shell node2
multipass shell node3
```

也可以直接执行命令：

```bash
multipass exec node1 -- <command>
multipass exec node2 -- <command>
multipass exec node3 -- <command>
```

## G3/G4-A Docker Swarm 基线与当前状态

2026-07-27 已在三个节点完成以下本地验证基线：

- Docker Engine/CLI 固定为 `29.6.1`，apt 包版本为 `5:29.6.1-1~ubuntu.24.04~noble`；安装源为 Docker 官方 Ubuntu apt 仓库；三个节点的仓库 GPG key SHA-256 均为 `1500c1f56fa9e26b9b8f42452a553675796ade0807cdce11975eb98170b3a570`；
- Docker 日志驱动为 `json-file`，轮转上限为单文件 `10m`、保留 `3` 个文件；
- `node1`、`node2`、`node3` 均为 `manager + worker`，运行态为 `Ready/Reachable`；G4-B2b-2b-2 后当前由 `node2` 担任唯一 Leader；
- `postgres=true` 只在 `node1`，`redis=true` 只在 `node2`；`node3` 当前无数据服务 label；
- 内部 overlay network 为 `sub2api-local-app`，`attachable=true`；
- Redis 所在 `node2` 已通过 `/etc/sysctl.d/99-sub2api-redis.conf` 持久化 `vm.overcommit_memory=1`；`node1`/`node3` 当前不承载 Redis，保持系统默认值；
- 镜像平台使用 OCI 名称 `arm64`，Docker 29.6.1 的 Swarm placement 字段实测为 `aarch64`，部署配置已分别记录；
- `sub2api=true` 与 `caddy=true` 已依次添加到 `node1`、`node2`、`node3`；`postgres=true` 仍只在 node1，`redis=true` 仍只在 node2；
- node1 的本地人工发布入口使用 GoTask `3.50.0`；当前 Sub2API ARM64 归档 SHA-256 为 `43d69c5fa76eb0f3c809a97251f2eee3477c04cb5324d6449fea6b6bb67b1f6c`，Caddy 归档 SHA-256 为 `cc8f05e47661ca5b41998b884831abc8e126082cf9ed697cd82fdc56d9c92ff2`。

当前 service 状态：G1 至 G5 本地实施基线已完成；PostgreSQL/Redis 按固定 ARM64 digest 分别位于 `node1`/`node2`，保持 `1/1`，Sub2API/Caddy 为 `3/3`。Sub2API 运行态 restart condition 已最小修正为 `any`，Caddy/PostgreSQL/Redis 保持 `on-failure`；node1/PostgreSQL 同场景复测、node3 单副本 OOM 与隔离 migration 失败均通过。PostgreSQL/Redis 数据身份、Caddy storage、证书、三个入口和最终 `release:verify ENV=local` 均正常。

本地镜像身份：

| 组件 | 本地 tag | 归档 SHA-256 | node1/node2/node3 image ID |
| --- | --- | --- | --- |
| Sub2API | `sub2api-local/sub2api:v0.1.165-ext.3-arm64` | `43d69c5fa76eb0f3c809a97251f2eee3477c04cb5324d6449fea6b6bb67b1f6c` | `sha256:fd867fc19da56a25bae98930d2186159f3650a83cc5cefb99164ae4951f01a6f` |
| Caddy | `sub2api-local/caddy:v2.11.4-redis-v1.8.1-arm64` | `cc8f05e47661ca5b41998b884831abc8e126082cf9ed697cd82fdc56d9c92ff2` | `sha256:26a85a756bcbd9d2f94d9bc55e48fce85ee55cf181b6002a3c82e1292504b739` |

当前 Sub2API `v0.1.165-ext.3` 固定到版本提交 `6c859d2d83e03c49fb49a53e530932d7a6c789d7`，annotated tag 已创建并推送；tag object 为 `de000a7f6ed506b76b10384da8301dc18c485637`。source image ID 为 `sha256:03e01bbd24c1818ac1f8ad9ec6413969ed9e6e69a524cb2795f993ed756da6aa`，容器内 `/app/sub2api` SHA-256 为 `c6d73fc00d060cf1d04ae0ffc3f76796b1c679bd14205692ad3f73c63e4e8b65`；未上传 GHCR。已验证的 `v0.1.165-ext.2` 镜像仍保留在三个节点作为回滚输入；更早的 `ext.1` 只作为 G4-B1 实际回滚证据保留。

宿主机曾因本地 Docker context 元数据缺失而无法直接访问 Swarm。恢复时只将宿主机既有 SSH 公钥加入 node1，并重建 `sub2api-local=ssh://ubuntu@192.168.252.2`；没有启用密码 SSH、没有暴露 Docker TCP daemon，也没有改变 service。正式发布命令仍从 node1 的固定工作副本执行。

三个 Caddy 入口共用的 Local CA 根证书 SHA-256 指纹为 `1C:F3:6C:A9:FF:B0:AE:B9:25:3E:B0:47:95:D4:76:5A:F0:41:B8:EE:3A:B7:7A:07:58:E4:F9:7A:89:93:A2:CB`。三个入口呈现的叶证书 subject 均为空、关键 SAN 均为 `DNS:sub2api.test`，serial 均为 `6A756405F963CC3B7D3310DCAF348F5B`，SHA-256 指纹均为 `40:FD:12:CF:C3:3C:C4:B8:45:80:75:AD:1F:09:91:C1:4E:A2:5D:FA:50:C5:F1:C2:3E:5C:C1:D5:A6:F3:15:7A`。G4-B2a 已证明单个 Caddy task 从正常共享 storage 重启后仍读取相同证书体系；G4-B2b-1 已证明 Redis 短时不可用期间既有 Caddy task 仍可完成 TLS 握手。Redis 不可用时 Caddy 冷启动和续期协调仍未验证。

本地 Stack 以 host mode 发布 Sub2API `8080` 供同节点 Caddy 访问，因此该端口也可从 Multipass 宿主机到达。本次测试环境已明确接受该安全例外；生产准入前必须通过防火墙或等价网络约束禁止绕过 Caddy。

该基线只用于同一台 macOS 宿主机上的编排验证，不证明跨物理故障域高可用。

## S4-B 非破坏性专项记录

2026-07-27 在不停止 task、节点、PostgreSQL 或 Redis，不制造 OOM、失败发布或实际回滚的边界内完成以下验证：

- node1 登录签发的 access JWT 可在三个节点使用；refresh token 在 node2 轮换后，新 access token 可在 node3 使用，旧 refresh token 被拒绝；node1 注销后，新 refresh token 在 node2 也被拒绝；
- 三个节点返回相同用户、分组、API Key 列表、`gpt-4o` 模型价格和版本 `0.1.165-ext.2`；一个临时 API Key 从 node1 创建后可在 node2/node3 读取，从 node2 删除后三个节点均不再可用，测试实体已清理；
- 三个 Caddy 的管理 QPS WebSocket 均完成 `101 Switching Protocols`；未携带 API Key 的 `/v1/models`、`/v1/responses`、`/v1/messages` 在三个节点返回一致 401；
- 当前正式数据中没有 Provider 账户、Provider API Key 或 Scheduled Test plan，因此 OAuth、SSE/OpenAI WebSocket、生图 limiter、Batch job lock、Scheduled Test、expiry、计费和 migration 使用协议级、race 或隔离 integration harness 验证，没有为了测试增加真实 Provider 配置或制造费用；
- 正式数据库的 `schema_migrations` 为 236 条记录、236 个唯一 filename、0 个空 checksum、0 组重复 filename；三个节点 TOTP 状态均为 disabled，近 500 行 Sub2API 日志的 password、Bearer、refresh token、JWT/TOTP secret 等敏感模式命中为 0。

该记录只证明 S4-B 非破坏性专项；后续滚动与实际回滚已由 G4-B1/S4-C 单独完成，单 task、单节点和 Caddy 重启恢复已由 G4-B2a 单独完成。三个正式 task 同时替换、双协调后端同时故障、TOTP 启用后的跨节点行为和证书续期在本地实施基线中仍未验证，继续作为生产前专项。

## G4-B1/S4-C 滚动与回滚记录

2026-07-27 已完成受控滚动、失败暂停、实际回滚和模型价格 Config 回滚：

- 历史 `ext.1` 镜像通过 source image ID、归档 SHA-256 和 node image ID 三重校验后加载到三个节点；GoTask 实际完成 `ext.2 -> ext.1 -> ext.2`。最终三个容器均使用 image ID `sha256:bb638caa30eac89bf8bb5ee6395361f941f83fc0f810150249901ba896561703`，`/app/sub2api` SHA-256 均为 `04bb1b3d8a39012a0c4e5135a950fd862b7171925b81abed70d54cbb63b5739c`；
- 单独滚动 Sub2API 镜像时，97 个逐秒样本最大同时失败入口数为 1；单独滚动应用 `/ready` healthcheck 时，92 个样本最大为 1；单独滚动 Caddy upstream health 时，73 个样本均为三个入口 200；
- 同一 Stack 同时改变 Sub2API healthcheck 与 Caddy upstream health 时，两个 service 独立并行滚动，94 个样本中有 1 个样本同时两个入口失败，但始终至少保留一个入口。因此关联变更必须串行：先使用新旧版本共同支持的健康路径滚动应用，再滚动应用 healthcheck，最后滚动 Caddy upstream health；
- 一个临时错误 app-config Secret 使首个新 task 不能连接数据库，Swarm 进入 `paused`，增强后的 `release:verify` 返回失败；恢复正式 `app-config-v001` 后 rollout completed，临时 Secret 已删除；
- 模型价格 Config 使用语义相同的临时内容哈希对象完成更新与回滚，前后镜像不变；最终重新引用 `sub2api-local-model-pricing-139de8a906ce`，临时 Config 已删除；
- 最终 Sub2API/Caddy 为 `3/3`，PostgreSQL/Redis 为 `1/1`，三个入口 `/ready=200`，应用 healthcheck 和 Caddy upstream health 均为 `/ready`，没有 `s4c` 临时对象残留。

该记录不覆盖 task kill、节点停止、Redis/PostgreSQL 中断、OOM、三个正式 task 同时替换或 TLS storage 恢复；其中低风险 task/节点/Caddy 重启已由后续 G4-B2a 单独完成。

## G4-B2a/S4-D 低风险故障记录

2026-07-27 已完成单 Sub2API task、node3 manager 和单 Caddy task 的受控故障恢复：

- node3 上单个 Sub2API task 被强制结束后约 10.6 秒恢复；采样期间 node1/node2 始终返回 200，仅 node3 短暂返回 502/503；
- node3 停止后在 15 秒内进入 `Down/Unreachable`，node1/node2 保持 manager quorum 和各一个 global task，Sub2API/Caddy 为 `2/3`，PostgreSQL/Redis 为 `1/1`，没有在剩余节点补第二副本；node3 重新启动后约 24 秒恢复 Ready/Reachable 和本机两个 global task；
- node3 上单个 Caddy task 被强制结束后约 5.2 秒恢复；采样期间 node1/node2 始终返回 200，仅 node3 短暂不可达；重启前后叶证书 serial/指纹一致，Redis DB 1 的 Caddy storage key 数保持 15、key name set SHA-256 保持 `8c59bdb6c96e954ab0b28d80c55c18c09e7ad3ed57992d7231c7a67c91a72ca8`，新 task 日志未出现证书签发事件；
- 最终 `release:verify ENV=local` 通过，Sub2API/Caddy 恢复 `3/3`，PostgreSQL/Redis 保持 `1/1`，三个 manager 均 Ready。

该记录不覆盖 Redis/PostgreSQL/数据节点中断、OOM、受控 migration 失败、证书续期协调或生产故障域。

## G4-B2b-1 Redis 中断恢复记录

2026-07-27 在不修改 service spec、不重启 Caddy 且不停止 node2 的边界内，将 Redis 容器 `17de281ecc86` 暂停约 25.05 秒后恢复：

- 暂停期间三个 Sub2API 均为直连 `/health=200`、`/ready=503`；三个 HTTPS 入口可完成 TLS 握手并返回 503，符合 Caddy active health 与依赖异常不误报 ready 的预期；
- 解除暂停后 Redis 约 0.09 秒恢复 `PONG`，约 1.0 秒后的完整采样中三个直连 `/ready` 和 HTTPS 入口均恢复 200，Docker health 约 10.1 秒后恢复 healthy；Redis task `qfhw450m6d8e30ambjwsi6k4n` 和 container ID 均未变化；
- 三个 Caddy task 未重启，证书 serial/指纹不变；Redis DB 1 的 Caddy storage key 数保持 15，key name set SHA-256 保持 `8c59bdb6c96e954ab0b28d80c55c18c09e7ad3ed57992d7231c7a67c91a72ca8`，没有观察到证书签发或 storage error；
- 最终 `release:verify ENV=local` 通过，Sub2API/Caddy 为 `3/3`，PostgreSQL/Redis 为 `1/1`。应用 DB 0 包含动态 TTL/锁/缓存 key，瞬时 key 数不作为数据完整性校验。

该记录只覆盖同一 Redis 进程的短时不可用与恢复，不覆盖 Redis 进程重启、AOF 重放、数据卷恢复、Redis 不可用时 Caddy 冷启动、真实 OAuth 事务、证书续期、PostgreSQL/数据节点中断或生产故障域。

## G4-B2b-2a PostgreSQL 中断恢复记录

2026-07-27 首次在 `ext.2` 上将 PostgreSQL 容器 `81c5e2921ae8` 暂停约 25.02 秒后恢复；该次 readiness 门槛未通过：

- 暂停期间三个 Sub2API 均保持直连 `/health=200`，但直连 `/ready` 在多轮 4 秒客户端期限内超时为 `000`，没有返回预期的 503；Caddy active health 在过渡后让三个 HTTPS 入口稳定返回 503，外层入口没有误报 ready；
- 解除暂停后 PostgreSQL 约 0.25 秒恢复 `pg_isready`，三个直连 `/ready` 恢复 200，三个 HTTPS 入口约 3.3 秒内全部恢复 200，Docker health 约 10.0 秒后恢复 healthy；
- PostgreSQL task `b5ysani4aye7gl2gbpxwv03v6`、container ID、volume `sub2api-local_postgres_data`、三个 Sub2API/Caddy task 均未变化；`schema_migrations` 恢复前后均为 236 条、236 个唯一 filename、0 个空 checksum；
- 最终 `release:verify ENV=local` 通过，Sub2API/Caddy 为 `3/3`，PostgreSQL/Redis 为 `1/1`，Sub2API panic/fatal 为 0。

该失败证据触发了严格限制在 `backend/extends/lifecycle/manager.go` 与 `manager_test.go` 的单 in-flight probe 与 caller 硬超时修补。修补后的 `v0.1.165-ext.3-arm64` 已按 source image ID、归档 SHA-256 和 node image ID 三重校验分发并滚动部署到三个节点，活动清单提交为 `3608d6c7b`。

同日复测时，PostgreSQL 容器在退出 trap 保护下从 `05:24:39.773162483 UTC` 暂停至 `05:25:04.865466972 UTC`，约 25.09 秒：

- 三个节点 `/health` 均保持 200；每节点连续三次、共九次直连 `/ready` 均返回 503，耗时约 2.0015–2.0653 秒；三个 HTTPS 入口均返回 503，原 4 秒 `000` 未复现；
- 解除暂停后约 15.75 秒取得首个完整恢复样本，PostgreSQL 已 accepting/healthy，三个直连 `/health`、`/ready` 与 HTTPS 均恢复 200；
- PostgreSQL task `b5ysani4aye7gl2gbpxwv03v6`、容器 `81c5e2921ae8`、volume `sub2api-local_postgres_data`、三个 Sub2API task 和三个 Caddy task 均未替换；`schema_migrations` 保持 `236/236/0`，近 10 分钟 Sub2API panic/fatal 为 0，最终 `release:verify ENV=local` 通过。

因此 `G4-B2b-2a` 只在“同一 PostgreSQL 容器短时暂停/恢复”的范围内通过；不覆盖 PostgreSQL 进程重启、volume 重挂载、数据节点停止、备份恢复、OOM、migration 失败或生产故障域。

## G4-B2b-2b 数据节点故障执行前只读基线

2026-07-27 已完成只读审查，没有停止节点或服务：

- node1/node2/node3 均为 `Ready/Active` manager，node1 当前为 Leader，quorum 为 2；已确认 node2 可通过 `multipass exec node2 -- docker node ls` 读取 Swarm，因此 node1 停止期间不依赖指向 node1 的 `sub2api-local` Docker context；
- PostgreSQL 当前 task/container 为 `t4ns8vvywx85`/`4a50cd8f4a12`，只允许位于 node1。`sub2api-local_postgres_data` 为 local volume，创建时间 `2026-07-27T00:28:23+08:00`，Mountpoint `/var/lib/docker/volumes/sub2api-local_postgres_data/_data`，device/inode `2049/302196`；`system_identifier=7666874411637911585`，migration 为 `236/236/0/0`（总数/唯一 filename/null checksum/空 checksum）；
- Redis 当前 task/container 为 `qfhw450m6d8e`/`17de281ecc86`，只允许位于 node2。`sub2api-local_redis_data` 为 local volume，创建时间 `2026-07-27T00:14:54+08:00`，Mountpoint `/var/lib/docker/volumes/sub2api-local_redis_data/_data`，device/inode `2049/299241`；`PONG`、RDB/AOF 均正常，DB 1 为 15 个 Caddy storage key，key name set SHA-256 为 `8c59bdb6c96e954ab0b28d80c55c18c09e7ad3ed57992d7231c7a67c91a72ca8`；
- 三个直连 `/health`、`/ready` 均为 200；三个叶证书 serial/指纹仍为 `6A756405F963CC3B7D3310DCAF348F5B` / `40:FD:12:CF:C3:3C:C4:B8:45:80:75:AD:1F:09:91:C1:4E:A2:5D:FA:50:C5:F1:C2:3E:5C:C1:D5:A6:F3:15:7A`；`release:verify ENV=local` 通过；
- 已确认 `/usr/local/bin/multipass` 与 `/usr/bin/nohup` 可用，未来每个场景都必须在停止前建立 60 秒 auto-start watchdog 与退出/信号恢复 trap。

审查把实际故障拆为 `G4-B2b-2b-1` node2/Redis 与 `G4-B2b-2b-2` node1/PostgreSQL。node2 已执行并通过；node1 首次执行因当时 30 秒门槛和 `on-failure` 配置缺口未通过，历史结论保留。最小环境恢复、配置层复盘、静态修正、运行态应用和修订门槛复测现均已通过；完整命令、自动恢复、停止门槛和实测证据见 [`GoTask-runbook.md`](./GoTask-runbook.md) 第 7.4 节。

该审查只为普通受控关机/原虚拟磁盘恢复做准备，不覆盖 `--force`、断电、宿主机崩溃、磁盘损坏、VM 重建、跨节点/备份恢复、自动故障转移、DNS 摘除、生产 HA 或 RPO/RTO。

## G4-B2b-2b-1 node2/Redis 数据节点停止/恢复记录

2026-07-27 已在普通 `multipass stop/start node2`、60 秒 auto-start watchdog 和退出/信号恢复 trap 保护下完成：

- 首次窗口确认 node2 不可达时，node1/node3 `/health=200`，直连 `/ready=503` 约 2.03–3.00 秒，HTTPS `/ready=503`，node1 保持 Leader；因恢复早于 Swarm task 汇总收敛，该窗口只保留为入口证据。完成全部恢复和数据复核后，在同一授权范围内复测；
- 复测时 node2 在 15 秒内进入 `Down/Unreachable`，node1/node3 保持 quorum；两个存活节点 `/health=200`、直连 `/ready=503` 约 3.00 秒、HTTPS 503，node2 入口不可达；Redis 新 task `qv9imdixga3m4mryny8arnu2a` 无 NODE，并明确因唯一 placement 无可用节点而 Pending，没有漂移；
- 故障期 `docker service ls` 显示 Sub2API/Caddy `3/2`、Redis `1/1`，原因是不可达节点旧 task 保留最后已知 `Running` 且 global desired 数已变为两个可用节点；实际验收应读 task-level desired/current state、NODE、调度错误和入口状态；
- 复测开始 35 秒时人工启动 node2，49 秒时返回，watchdog 未触发。恢复后的 Redis task/container 为 `qv9imdixga3m4mryny8arnu2a`/`9cf548417e10`，从原 AOF 加载约 0.034 秒；volume `sub2api-local_redis_data` 的创建时间、Mountpoint 和 device/inode `2049/299241` 均不变；
- Redis 恢复 `PONG`，RDB/AOF 正常；Caddy DB 1 仍为 15 个 key，摘要仍为 `8c59bdb6c96e954ab0b28d80c55c18c09e7ad3ed57992d7231c7a67c91a72ca8`；三个证书 serial/指纹、PostgreSQL `system_identifier=7666874411637911585` 和 migration `236/236/0/0` 均不变；
- 三个 manager、Sub2API/Caddy `3/3`、PostgreSQL/Redis `1/1` 和三个入口全部恢复，相关日志无 panic/fatal/corruption、新证书签发或 Redis storage error，最终 `release:verify ENV=local` 通过。

该结果只证明 node2 的普通受控关机、原虚拟磁盘和原 local volume 恢复，不证明 `--force`、断电、磁盘损坏、VM 重建、跨节点/备份恢复、自动故障转移、DNS 摘除或生产 HA。

## G4-B2b-2b-2 node1/PostgreSQL 复审与执行记录

2026-07-27 已完成复审，没有停止节点或服务：

- 三个 manager 均为 `Ready/Active`，node1 当前为唯一 Leader；宿主机 Docker context `sub2api-local` 固定为 `ssh://ubuntu@192.168.252.2`，node1 停止后不可用；
- node2 可使用原生 `/usr/bin/docker` 管理 Swarm，`/usr/bin/timeout`、`curl`、`openssl` 均可用；但 node2 没有 `task`、部署目录或本地 CA，因此故障期只使用有界 Docker CLI、直连入口和证书 serial/指纹取证，完整 `release:verify` 必须等 node1 恢复后执行；
- PostgreSQL service placement 为唯一 `postgres=true` 节点及 `aarch64`，当前 task/container 为 `t4ns8vvywx85wdwon1dksg524`/`4a50cd8f4a12`。named volume `sub2api-local_postgres_data` 的创建时间、Mountpoint 与 device/inode `2049/302196` 均已记录；`pg_isready`、`system_identifier=7666874411637911585` 和 migration `236/236/0/0` 通过；
- Redis 继续位于 node2 且 `PONG`；node2/node3 直连和 HTTPS 当前均为 200，证书 serial/指纹与既有基线一致；正确入口上的 `release:verify ENV=local` 通过；
- node1 使用硬编码的专用 60 秒 watchdog + `EXIT/INT/TERM/HUP` trap 命令，不能编辑 node2 模板。只有 `multipass start node1` 成功、状态为 `Running` 且 `multipass exec node1 -- true` 成功后才能撤销恢复保护；
- 配置修正应用后另行授权的复测使用拆分门槛：stop 返回后 30 秒内 node2/node3 保持 quorum 且只有一个 Leader；50 秒内 PostgreSQL 新 desired task 无 NODE、因唯一 placement 无可用节点而 Pending，且不得漂移。任一门槛失败即恢复并判失败；Pending 一出现即人工恢复，不等待 watchdog。保留 60 秒 watchdog。HTTPS 临时 `curl -k` 必须同时核对预先记录的 serial/指纹，不能替代恢复后的 CA 验证。该规则不追认历史执行通过。

只读复审结论为通过，两个文档级阻断项已关闭。随后取得独立授权并实际执行；执行没有新增代码、脚本、任务、服务、daemon、控制面或其他实体。

实际执行结果为 **未通过，环境仅部分恢复**：

- `06:52:03Z` 建立 60 秒 watchdog/trap 后普通停止 node1，stop 于 `06:52:07Z` 返回；node2 约 12 秒后成为唯一 Leader，约 17 秒时 node2/node3 均为 Ready，但截至 30 秒 PostgreSQL 仍显示 node1 上旧 task，没有形成无 NODE/Pending task，故立即恢复；
- node1 故障期不可达；node2/node3 `/health=200`、直连 `/ready=503` 约 2.04–2.08 秒、HTTPS `/ready=503`，证书 serial/指纹不变。`06:52:43Z` 人工启动 node1，确认 VM Running 和来宾可执行后撤销保护，watchdog 未触发；
- PostgreSQL task/container 变为 `zf8yvth1nrkna4iehx79g6qkj`/`8139962ea6fc`，但继续使用原 `sub2api-local_postgres_data`；创建时间、Mountpoint、device/inode `2049/302196`、`system_identifier=7666874411637911585` 和 migration `236/236/0/0` 均不变。Redis、Caddy storage、证书与三个 manager 也无数据或身份漂移；
- node2/node3 的 Sub2API task 在依赖不可用期间变为 unhealthy，并分别以 `exit 0`、task `Complete` 结束；`restart_policy.condition=on-failure` 没有自动重建。300 秒后 Caddy 仍为 `3/3`、PostgreSQL/Redis 为 `1/1`，但 Sub2API 仅 node1 `1/1`；node1 直连/HTTPS `/ready=200`，node2/node3 直连 `:8080` 不可达、HTTPS `/ready=503`，`release:verify ENV=local` 失败；
- 随后独立授权的最小恢复只对 `sub2api-local_sub2api` 执行一次 force-update；ForceUpdate generation 从 0 增至 1，其他已登记 service 字段不变。rollout 于 `07:04:22Z–07:05:49Z` 串行完成，新 task 为 node1 `sfb1tefnklak28qjnniczxgb3`、node2 `i8f63oj42ylw3a2z3gwjzax70`、node3 `q0jmzdsi8ickapy8ixthzz695`，均为 healthy；
- 恢复后 Sub2API/Caddy `3/3`、PostgreSQL/Redis `1/1`，三个直连与 HTTPS `/health`、`/ready` 均为 200，`release:verify ENV=local` 通过。PostgreSQL/Redis 数据身份、Caddy storage、证书与日志门槛不变，没有重部署 Stack 或触碰其他 service/节点；
- 随后的配置层只读复盘已通过：stop 返回后约 12 秒形成 Leader、约 17 秒两个存活 manager Ready，但 node1 heartbeat 到约 40 秒才过期并触发 PostgreSQL 新 desired task，证明原统一 30 秒门槛过早；node2/node3 应用 task 均以 `exit 0/Complete` 结束，原 `condition=on-failure` 无法补齐。最终只把 Sub2API condition 改为 `any`，并在既有 `validate:stack` 增加四 service 渲染断言；`/ready`、healthcheck 其余参数、`delay/max_attempts/window`、其他 service 与全部 Go 代码不变；
- 有效复测在 stop 返回后 0 秒确认 node2/node3 quorum 与唯一 Leader，15 秒出现无 NODE/Pending 的 PostgreSQL task 并立即恢复；两个存活入口 `/health=200`、直连 `/ready=503` 约 2.01–2.04 秒。node1 恢复后 12 秒内全栈与三个入口恢复，Sub2API task 自动补齐，最终 `release:verify` 和全部数据/TLS 不变量通过；
- `G4-B2c` 中 node3 单副本 cgroup OOM 达到 2 GiB、原容器 `OOMKilled=true/exit=137`，node1/node2 全程 200，node3 约 11 秒恢复。隔离 migration checksum 故障的临时 task 以 exit 1 失败且未 ready，正式 migration 保持 `236/236/0/0`，临时 service/Secret/数据库已全部删除。
