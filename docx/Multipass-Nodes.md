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

正式 Sub2API `v0.1.165-ext.2` 固定到 commit `9aca50a8fd1ad34de6ef6ecf08eb58800a19fa89`，source image ID 为 `sha256:d6f956d592de70534e0c94fcff4199515dda555acc6f6ccef6405099daff5539`。node1 更新的 Swarm `PreviousSpec` 仍指向 `v0.1.165-ext.1`，对应旧 image ID `sha256:658b62d53062a22140670a40622b65f69432c7f32293113e2960c74b826e1e04` 已保留在 node1；本轮只核实回滚目标，没有实际执行 rollback。

三个 Caddy 入口共用的 Local CA 根证书 SHA-256 指纹为 `1C:F3:6C:A9:FF:B0:AE:B9:25:3E:B0:47:95:D4:76:5A:F0:41:B8:EE:3A:B7:7A:07:58:E4:F9:7A:89:93:A2:CB`。三个入口呈现的叶证书 subject 均为空、关键 SAN 均为 `DNS:sub2api.test`，serial 均为 `6A756405F963CC3B7D3310DCAF348F5B`，SHA-256 指纹均为 `40:FD:12:CF:C3:3C:C4:B8:45:80:75:AD:1F:09:91:C1:4E:A2:5D:FA:50:C5:F1:C2:3E:5C:C1:D5:A6:F3:15:7A`。这证明当前三个入口读取到同一证书体系，但不替代后续的续期、Redis 中断或 Caddy 重启恢复演练。

本地 Stack 以 host mode 发布 Sub2API `8080` 供同节点 Caddy 访问，因此该端口也可从 Multipass 宿主机到达。本次测试环境已明确接受该安全例外；生产准入前必须通过防火墙或等价网络约束禁止绕过 Caddy。

该基线只用于同一台 macOS 宿主机上的编排验证，不证明跨物理故障域高可用。

## S4-B 非破坏性专项记录

2026-07-27 在不停止 task、节点、PostgreSQL 或 Redis，不制造 OOM、失败发布或实际回滚的边界内完成以下验证：

- node1 登录签发的 access JWT 可在三个节点使用；refresh token 在 node2 轮换后，新 access token 可在 node3 使用，旧 refresh token 被拒绝；node1 注销后，新 refresh token 在 node2 也被拒绝；
- 三个节点返回相同用户、分组、API Key 列表、`gpt-4o` 模型价格和版本 `0.1.165-ext.2`；一个临时 API Key 从 node1 创建后可在 node2/node3 读取，从 node2 删除后三个节点均不再可用，测试实体已清理；
- 三个 Caddy 的管理 QPS WebSocket 均完成 `101 Switching Protocols`；未携带 API Key 的 `/v1/models`、`/v1/responses`、`/v1/messages` 在三个节点返回一致 401；
- 当前正式数据中没有 Provider 账户、Provider API Key 或 Scheduled Test plan，因此 OAuth、SSE/OpenAI WebSocket、生图 limiter、Batch job lock、Scheduled Test、expiry、计费和 migration 使用协议级、race 或隔离 integration harness 验证，没有为了测试增加真实 Provider 配置或制造费用；
- 正式数据库的 `schema_migrations` 为 236 条记录、236 个唯一 filename、0 个空 checksum、0 组重复 filename；三个节点 TOTP 状态均为 disabled，近 500 行 Sub2API 日志的 password、Bearer、refresh token、JWT/TOTP secret 等敏感模式命中为 0。

该记录只证明 S4-B 非破坏性专项。最小滚动排空、三个正式 task 同时替换、双协调后端同时故障、TOTP 启用后的跨节点行为、实际回滚、TLS 恢复和故障矩阵均未执行，仍受 `G4-B` 或后续单独授权约束。
