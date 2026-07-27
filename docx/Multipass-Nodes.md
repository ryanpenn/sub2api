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

## G3/G4-A Docker Swarm 基线

2026-07-27 已在三个节点完成以下本地验证基线：

- Docker Engine/CLI 固定为 `29.6.1`，apt 包版本为 `5:29.6.1-1~ubuntu.24.04~noble`；安装源为 Docker 官方 Ubuntu apt 仓库；三个节点的仓库 GPG key SHA-256 均为 `1500c1f56fa9e26b9b8f42452a553675796ade0807cdce11975eb98170b3a570`；
- Docker 日志驱动为 `json-file`，轮转上限为单文件 `10m`、保留 `3` 个文件；
- `node1`、`node2`、`node3` 均为 `manager + worker`，运行态为 `Ready/Reachable`；`node1` 为唯一 Leader；
- `postgres=true` 只在 `node1`，`redis=true` 只在 `node2`；`node3` 当前无数据服务 label；
- 内部 overlay network 为 `sub2api-local-app`，`attachable=true`；
- Redis 所在 `node2` 已通过 `/etc/sysctl.d/99-sub2api-redis.conf` 持久化 `vm.overcommit_memory=1`；`node1`/`node3` 当前不承载 Redis，保持系统默认值；
- 镜像平台使用 OCI 名称 `arm64`，Docker 29.6.1 的 Swarm placement 字段实测为 `aarch64`，部署配置已分别记录；
- `sub2api=true` 与 `caddy=true` 已依次添加到 `node1`、`node2`、`node3`；`postgres=true` 仍只在 node1，`redis=true` 仍只在 node2；
- node1 的本地人工发布入口使用 GoTask `3.50.0`；当前 Sub2API ARM64 归档 SHA-256 为 `3e1c69b1d96417acbd615ca7d48b8dbda60f070e65ccb6c0f80c59a095acae70`，Caddy 归档 SHA-256 为 `cc8f05e47661ca5b41998b884831abc8e126082cf9ed697cd82fdc56d9c92ff2`。

当前 service 状态：PostgreSQL/Redis 均按固定 ARM64 digest 运行，分别位于 `node1`/`node2`，保持 `1/1`；Sub2API/Caddy 使用本地归档镜像在三个节点各运行一个 task，均为 `3/3`。PostgreSQL `pg_isready` 通过且没有发布端口，Redis 通过官方 entrypoint 降权后主进程使用 `redis` 用户。三个 Caddy 入口的 `https://sub2api.test/ready` 均返回 JSON 200，证明各自本机 Sub2API 可同时连接共享 PostgreSQL/Redis；在线更新检查入口均被 Caddy 拒绝为 403。历史失败 task 只作为证据保留，验收以当前 desired-state task 数和健康状态为准。

本地镜像身份：

| 组件 | 本地 tag | 归档 SHA-256 | node1/node2/node3 image ID |
| --- | --- | --- | --- |
| Sub2API | `sub2api-local/sub2api:v0.1.165-ext.2-arm64` | `3e1c69b1d96417acbd615ca7d48b8dbda60f070e65ccb6c0f80c59a095acae70` | `sha256:bb638caa30eac89bf8bb5ee6395361f941f83fc0f810150249901ba896561703` |
| Caddy | `sub2api-local/caddy:v2.11.4-redis-v1.8.1-arm64` | `cc8f05e47661ca5b41998b884831abc8e126082cf9ed697cd82fdc56d9c92ff2` | `sha256:26a85a756bcbd9d2f94d9bc55e48fce85ee55cf181b6002a3c82e1292504b739` |

正式 Sub2API `v0.1.165-ext.2` 固定到 commit `9aca50a8fd1ad34de6ef6ecf08eb58800a19fa89`，source image ID 为 `sha256:d6f956d592de70534e0c94fcff4199515dda555acc6f6ccef6405099daff5539`。旧版本 `v0.1.165-ext.1` 的 node image ID 为 `sha256:658b62d53062a22140670a40622b65f69432c7f32293113e2960c74b826e1e04`；G4-B1 已按历史归档 SHA-256 把旧镜像加载到三个节点并完成实际回滚，最终正式 service 已重新恢复为 `ext.2`。

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

该记录只证明 S4-B 非破坏性专项；后续滚动与实际回滚已由 G4-B1/S4-C 单独完成，单 task、单节点和 Caddy 重启恢复已由 G4-B2a 单独完成。三个正式 task 同时替换、双协调后端同时故障、TOTP 启用后的跨节点行为、证书续期及其余故障矩阵仍未执行，继续受后续单独授权约束。

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

2026-07-27 在不停止 node1、不修改 volume 或 service spec 的边界内，将 PostgreSQL 容器 `81c5e2921ae8` 暂停约 25.02 秒后恢复。本项执行完成，但 readiness 门槛未通过：

- 暂停期间三个 Sub2API 均保持直连 `/health=200`，但直连 `/ready` 在多轮 4 秒客户端期限内超时为 `000`，没有返回预期的 503；Caddy active health 在过渡后让三个 HTTPS 入口稳定返回 503，外层入口没有误报 ready；
- 解除暂停后 PostgreSQL 约 0.25 秒恢复 `pg_isready`，三个直连 `/ready` 恢复 200，三个 HTTPS 入口约 3.3 秒内全部恢复 200，Docker health 约 10.0 秒后恢复 healthy；
- PostgreSQL task `b5ysani4aye7gl2gbpxwv03v6`、container ID、volume `sub2api-local_postgres_data`、三个 Sub2API/Caddy task 均未变化；`schema_migrations` 恢复前后均为 236 条、236 个唯一 filename、0 个空 checksum；
- 最终 `release:verify ENV=local` 通过，Sub2API/Caddy 为 `3/3`，PostgreSQL/Redis 为 `1/1`，Sub2API panic/fatal 为 0。

当前证据表明 Caddy/Swarm 外层 probe 能 fail-closed，但应用的 PostgreSQL `PingContext` 未在既定 2 秒预算内返回。完成失败证据审核和最小修补授权前，不重复 PostgreSQL 故障注入；本记录也不覆盖 PostgreSQL 进程重启、volume 重挂载、数据节点停止、备份恢复、OOM、migration 失败或生产故障域。
