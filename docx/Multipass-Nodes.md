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
- `sub2api=true` 与 `caddy=true` 尚未添加；按单副本基线顺序，需等待数据服务健康且一次性 bootstrap 成功后才添加到 `node1`；
- node1 的本地人工发布入口使用 GoTask `3.50.0`，ARM64 归档 SHA-256 为 `ee67e7d999a4a70711bff1946c70bf76628012c91d9be55626ee90ba976897da`。

当前数据 service 状态：PostgreSQL/Redis 均按固定 ARM64 digest 运行，分别位于 `node1`/`node2`，task 均为 `1/1` 且 health 为 healthy；PostgreSQL `pg_isready` 通过且没有发布端口，Redis 通过官方 entrypoint 降权后主进程使用 `redis` 用户。Sub2API/Caddy 尚无应用 label，按阶段 2 顺序保持 `0/0`。Redis 修复前的失败 task 仅作为历史证据保留，验收以当前 desired-state task 数和健康状态为准。

该基线只用于同一台 macOS 宿主机上的编排验证，不证明跨物理故障域高可用。
