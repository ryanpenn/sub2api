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

## G3 Docker Swarm 基线

2026-07-27 已在三个节点完成以下本地验证基线：

- Docker Engine/CLI 固定为 `29.6.1`，apt 包版本为 `5:29.6.1-1~ubuntu.24.04~noble`；安装源为 Docker 官方 Ubuntu apt 仓库；三个节点的仓库 GPG key SHA-256 均为 `1500c1f56fa9e26b9b8f42452a553675796ade0807cdce11975eb98170b3a570`；
- Docker 日志驱动为 `json-file`，轮转上限为单文件 `10m`、保留 `3` 个文件；
- `node1`、`node2`、`node3` 均为 `manager + worker`，运行态为 `Ready/Reachable`；`node1` 为唯一 Leader；
- `postgres=true` 只在 `node1`，`redis=true` 只在 `node2`；`node3` 当前无数据服务 label；
- 内部 overlay network 为 `sub2api-local-app`，`attachable=true`；
- Redis 所在 `node2` 已通过 `/etc/sysctl.d/99-sub2api-redis.conf` 持久化 `vm.overcommit_memory=1`；`node1`/`node3` 当前不承载 Redis，保持系统默认值；
- 镜像平台使用 OCI 名称 `arm64`，Docker 29.6.1 的 Swarm placement 字段实测为 `aarch64`，部署配置已分别记录；
- `sub2api=true` 与 `caddy=true` 已在一次性 bootstrap 成功后仅添加到 `node1`；`node2`/`node3` 留待阶段 4；
- node1 的本地人工发布入口使用 GoTask `3.50.0`，ARM64 归档 SHA-256 为 `ee67e7d999a4a70711bff1946c70bf76628012c91d9be55626ee90ba976897da`。

当前 service 状态：PostgreSQL/Redis 均按固定 ARM64 digest 运行，分别位于 `node1`/`node2`；Sub2API/Caddy 使用本地归档镜像运行于 `node1`；四项 service 均为 `1/1`。PostgreSQL `pg_isready` 通过且没有发布端口，Redis 通过官方 entrypoint 降权后主进程使用 `redis` 用户。Sub2API 经 Caddy 的 `https://sub2api.test/health` 返回 200，管理员登录通过；Caddy 强制重建前后 Local CA SHA-256 指纹一致。Redis 修复前和 bootstrap 首次缺少临时目录的失败 task 仅作为历史证据保留，验收以当前 desired-state task 数和健康状态为准。

本地镜像身份：

| 组件 | 本地 tag | 归档 SHA-256 | node1 image ID |
| --- | --- | --- | --- |
| Sub2API | `sub2api-local/sub2api:v0.1.165-ext.1-arm64` | `150e648aeefec2cd541807bb726e9ca4b4c243f4f1cf639045d50ce49a51da39` | `sha256:658b62d53062a22140670a40622b65f69432c7f32293113e2960c74b826e1e04` |
| Caddy | `sub2api-local/caddy:v2.11.4-redis-v1.8.1-arm64` | `cc8f05e47661ca5b41998b884831abc8e126082cf9ed697cd82fdc56d9c92ff2` | `sha256:26a85a756bcbd9d2f94d9bc55e48fce85ee55cf181b6002a3c82e1292504b739` |

阶段 3 候选 `sub2api-local/sub2api:v0.1.165-ext.2-arm64` 已加载到 node1，但尚未更新正式 service。它固定到 commit `9aca50a8fd1ad34de6ef6ecf08eb58800a19fa89`，source image ID 为 `sha256:d6f956d592de70534e0c94fcff4199515dda555acc6f6ccef6405099daff5539`，归档 SHA-256 为 `3e1c69b1d96417acbd615ca7d48b8dbda60f070e65ccb6c0f80c59a095acae70`，node1 image ID 为 `sha256:bb638caa30eac89bf8bb5ee6395361f941f83fc0f810150249901ba896561703`。三个隔离容器的全新数据库并发 bootstrap 已通过，临时容器、数据库和 Redis DB 15 均已清理。

本地 Stack 以 host mode 发布 Sub2API `8080` 供同节点 Caddy 访问，因此该端口也可从 Multipass 宿主机到达。本次测试环境已明确接受该安全例外；生产准入前必须通过防火墙或等价网络约束禁止绕过 Caddy。

该基线只用于同一台 macOS 宿主机上的编排验证，不证明跨物理故障域高可用。
