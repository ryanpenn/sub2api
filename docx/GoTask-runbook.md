# GoTask 发布与运维手册

> 状态：设计手册，GoTask/Stack 文件尚未生成
> 适用范围：Sub2API Docker Swarm 本地 ARM64 验证与后续 AMD64 生产环境
> 基线日期：2026-07-26（Asia/Shanghai）

## 1. 目的与边界

本手册说明 GoTask 的基本使用方式，以及如何将它作为 Sub2API 集群的统一发布/运维 CLI 入口。最终架构仍为：

- Git 管理可审核的 Stack、Caddyfile、脱敏环境模板和发布契约；
- GoTask 将校验、部署、验证、回滚和日常运维命令组合成稳定入口；
- Docker Swarm 管理 service 的期望状态、调度、滚动更新和回滚；
- Caddy 在每个节点固定代理本机 Sub2API，不引入 Traefik 或 routing mesh 公网分流；
- 镜像构建/推送与集群部署分离，Manager 只部署经验证的固定 digest。

GoTask 不是长驻运维平台，不承担：

- Web UI、Agent、RBAC、审批和监控；
- Secret 存储、集群状态库或发布数据库；
- 跨操作者/跨 CI Job 的分布式锁；
- 跨多个 Swarm service 的原子交易；
- 节点的 SSH/Agent 远程管理。

因此，第一期由指定 Manager 上的单一操作者人工执行；生产阶段再用外部 CI/CD runner 串行调用相同 Task，复用 CI/CD 的审批和日志，不自建发布控制面。

## 2. 环境与术语

### 2.1 当前本地节点

| 节点 | IP | 架构 | 角色 |
| --- | --- | --- | --- |
| `node1` | `192.168.252.2` | `linux/arm64` | Caddy + Sub2API + PostgreSQL |
| `node2` | `192.168.252.3` | `linux/arm64` | Caddy + Sub2API + Redis |
| `node3` | `192.168.252.4` | `linux/arm64` | Caddy + Sub2API |

本地节点每台 4G 内存，仅使用缩小资源档验证功能、调度和限额语义，不用于推导生产容量。节点详细信息见 [`Multipass-Nodes.md`](./Multipass-Nodes.md)。

### 2.2 环境参数

| 变量 | 用途 | 示例 |
| --- | --- | --- |
| `ENV` | 目标环境 | `local`、`production` |
| `RELEASE` | fork 发布版本 | `v0.1.165-ext.1` |
| `NODE` | Swarm 节点名 | `node3`、`node4` |
| `SERVICE` | 查询日志/状态的 service | `sub2api`、`caddy` |
| `CONFIRM` | 高风险任务的非敏感确认字符串 | `bootstrap-sub2api` |

`CONFIRM` 不是密码。JWT/TOTP key、数据库/Redis 密码、Caddy storage encryption key 和 Provider 凭据不得放入 Task 变量、命令行、Git 或发布摘要，Task 只引用已创建的版本化 Swarm Secret 对象。

## 3. 安装与基本命令

### 3.1 安装

开发机 macOS 可使用官方 Homebrew tap：

```bash
brew install go-task/tap/go-task
```

Ubuntu/Debian 可使用 GoTask 官方 apt 仓库，也可从 GitHub Release 下载固定版本二进制。生产环境必须在配置中记录 GoTask 版本并校验官方 `task_checksums.txt` 中的 SHA-256，不直接执行未固定版本的远程安装脚本。

安装后检查：

```bash
task --version
task --help
```

本手册核对时的官方文档版本为 `v3.52.0`；实施时应再确认并固定项目实际采用版本。

### 3.2 运行任务

实施后，在 fork 根目录执行：

```bash
task --dir deploy/cluster --list-all
task --dir deploy/cluster validate:environment ENV=local
```

或先进入目录：

```bash
cd deploy/cluster
task --list-all
task validate:environment ENV=local
```

常用操作：

| 命令 | 用途 |
| --- | --- |
| `task --list-all` | 列出所有公开任务 |
| `task <namespace>:<task>` | 运行指定任务 |
| `task <task> KEY=value` | 显式传入变量 |
| `task --verbose <task>` | 输出详细执行信息，仅用于不包含敏感值的调试 |

不建议在生产使用 `--force` 绕过 `preconditions`。GoTask 的 `requires`/`preconditions` 是防误用机制，并不是不可绕过的安全边界。

## 4. Taskfile 最小组织方式

方案固定的最小目录为：

```text
deploy/cluster/
├── Taskfile.yml
├── taskfiles/
│   ├── validate.yml
│   ├── release.yml
│   └── ops.yml
├── scripts/
├── stacks/
└── env/
    ├── local-arm64/
    └── production-amd64/
```

根 `Taskfile.yml` 只做命名空间组合：

```yaml
version: '3'

includes:
  validate: ./taskfiles/validate.yml
  release: ./taskfiles/release.yml
  ops: ./taskfiles/ops.yml

set: [errexit, nounset, pipefail]
```

子 Taskfile 只提供稳定入口，复杂校验进入可测试脚本。例如 `taskfiles/validate.yml` 的示意形式：

```yaml
version: '3'

tasks:
  environment:
    desc: Validate target environment and Swarm identity
    requires:
      vars:
        - name: ENV
          enum: [local, production]
    cmds:
      - ./scripts/validate-environment.sh "{{.ENV}}"

  stack:
    desc: Validate rendered Stack before deployment
    requires:
      vars:
        - name: ENV
          enum: [local, production]
    cmds:
      - ./scripts/validate-stack.sh "{{.ENV}}"
```

上述只是结构示例，不代表脚本已存在。实际配置生成前，手册中的 Task 命令不可直接用于部署。

## 5. 首次部署服务

### 5.1 部署前提

首次部署前必须满足：

- 在指定 Manager 上执行，且 Docker context、Swarm 集群标识和 Manager quorum 校验通过；
- 已在 CI 中构建并推送与目标架构匹配的镜像；Manager 不负责编译镜像；
- Stack 使用平台对应的固定镜像 digest，不使用 `latest` 或仅固定 tag；
- 版本化 Swarm Config/Secret 已创建，Secret 不出现在 Git、Task 输出和命令历史中；
- 节点 label、placement constraint、资源 reservation/limit 与目标环境一致；
- 普通 Sub2API 副本固定 `AUTO_SETUP=false`；全新数据库仅允许一个受控 bootstrap 流程；
- PostgreSQL、Redis、Caddy 和 Sub2API 的持久目录与备份边界已经确认。

### 5.2 本地全新部署示例

```bash
cd deploy/cluster

task validate:environment ENV=local
task validate:stack ENV=local
task release:plan ENV=local RELEASE=v0.1.165-ext.1
task release:bootstrap ENV=local CONFIRM=bootstrap-sub2api
task release:apply ENV=local RELEASE=v0.1.165-ext.1
task release:verify ENV=local
task ops:status
```

命令语义如下：

1. `validate:environment` 校验 Docker context、Swarm 身份、节点架构、label、网络和必要对象；
2. `validate:stack` 渲染并校验 Stack，检查固定 digest、资源约束、placement 和 Config/Secret 引用；
3. `release:plan` 只展示将要变更的 image digest、Config/Secret 版本和 service，不执行修改；
4. `release:bootstrap` 仅用于全新数据库，必须显式确认，后续更新不得再次调用；
5. `release:apply` 执行 `docker stack deploy` 并等待滚动更新进入稳定状态；
6. `release:verify` 从每个节点的本机 Caddy 入口检查 `/ready`，并核对实际运行的 digest、Config/Secret 和 placement。

数据库 migration 仍由 Sub2API 启动过程执行，并通过 PostgreSQL advisory lock 串行化。第一期不新增独立 migration service/job，以避免引入额外实体；`bootstrap` 只负责全新数据库的首次初始化控制，不替代应用自身 migration。

### 5.3 本地逐节点验证

在自动化任务尚未实现前，可使用本地 CA 对每个节点做等价检查。以下证书路径为占位符：

```bash
curl --noproxy '*' \
  --resolve sub2api.test:443:192.168.252.2 \
  --cacert <caddy-local-ca.pem> \
  https://sub2api.test/ready

curl --noproxy '*' \
  --resolve sub2api.test:443:192.168.252.3 \
  --cacert <caddy-local-ca.pem> \
  https://sub2api.test/ready

curl --noproxy '*' \
  --resolve sub2api.test:443:192.168.252.4 \
  --cacert <caddy-local-ca.pem> \
  https://sub2api.test/ready
```

只检查 `docker service ls` 不足以判定发布成功；必须验证每个节点的真实入口、运行版本和共享依赖。

## 6. 更新与回滚

### 6.1 更新 Sub2API

例如从 `v0.1.165-ext.1` 更新到 `v0.1.165-ext.2`：

1. CI 构建、测试并推送 `linux/arm64` 和 `linux/amd64` 镜像；
2. 记录各平台不可变 digest，并更新对应环境的发布清单；
3. 在指定 Manager 上先生成 plan，再执行滚动更新；
4. 检查每个副本的 `/ready`、digest、日志及共享 PostgreSQL/Redis 连接。

```bash
cd deploy/cluster

task validate:environment ENV=local
task validate:stack ENV=local
task release:plan ENV=local RELEASE=v0.1.165-ext.2
task release:apply ENV=local RELEASE=v0.1.165-ext.2
task release:verify ENV=local
task ops:status
task ops:logs SERVICE=sub2api
```

应用更新默认只更新 Sub2API service，不隐式更新 PostgreSQL、Redis 或 Caddy。只有发布清单明确包含相应变更并通过独立校验时，才允许更新这些基础服务。

滚动更新不等于暂停整个集群：Swarm 按更新策略逐批替换副本，其余健康节点继续服务。但是当前 Caddy 固定代理本机 Sub2API，因此某节点上的副本被替换时，该节点入口可能短暂不可用。生产 DNS 尚无自动故障摘除，应通过合理的 `parallelism`、`delay`、健康检查和失败暂停策略降低影响。

### 6.2 更新模型价格等配置

模型价格缓存属于配置更新，不要求重新构建应用镜像。建议创建新的版本化 Swarm Config，更新 service 对 Config 的引用，再沿用相同的 `plan -> apply -> verify` 流程触发滚动更新。因为运行中的容器不会自动获取新 Config，仍需要替换相关 Sub2API task，但不需要全局停服。

### 6.3 回滚

```bash
task release:rollback ENV=local RELEASE=v0.1.165-ext.1
task release:verify ENV=local
task ops:status
```

回滚目标必须是发布记录中已验证的上一版本，包含旧 image digest 以及与其匹配的 Config/Secret 版本；不能只依赖临时执行 `docker service update --rollback`。如果数据库 migration 不向后兼容，应先停止自动回滚并人工评估，避免旧应用读取新 schema。

## 7. 服务上下线

### 7.1 “下线”的定义

日常维护中的下线是将某个节点退出业务流量并迁走其 Swarm task，不是执行 `docker stack rm`。第一期不提供通用卸载任务，也不允许运维任务顺带删除 volume、Secret、Config 或数据服务。

由于 Caddy 与 Sub2API 按节点配对部署，节点下线会同时减少一个公网入口和一个 Sub2API 副本。生产环境在 drain 前，应先人工移除该节点的 DNS A 记录，等待 DNS TTL 和已有连接消退；当前不实现 DNS 自动故障摘除。

### 7.2 下线普通业务节点

以不承载 PostgreSQL/Redis 的 `node3` 为例：

```bash
task ops:node-status NODE=node3
task ops:drain NODE=node3 CONFIRM=drain-node3
task ops:node-status NODE=node3
task ops:status
```

`ops:drain` 对应 `docker node update --availability drain`，会迁走该节点上的**所有** Swarm task，而不只是 Sub2API。执行前必须核对节点角色和实际 task。`node1`、`node2` 当前承载有状态服务，通用 `ops:drain` 应默认拒绝，只有完成备份并提供专门的数据服务迁移方案后才能操作。

### 7.3 重新上线普通业务节点

```bash
task ops:undrain NODE=node3 CONFIRM=undrain-node3
task release:verify ENV=local
task ops:status
```

`ops:undrain` 将节点 availability 恢复为 `active`。必须等 Caddy、Sub2API、TLS、共享依赖和 `/ready` 全部验证通过后，生产环境才能人工恢复该节点的 DNS A 记录。

整个 Sub2API 应用的下线属于独立变更，不复用节点 drain，也不删除 PostgreSQL/Redis。需要明确维护窗口、调用方行为、数据保护和恢复步骤后，再设计带确认门槛的专用任务。

## 8. 新增 Node4：增加第 4 个 Sub2API 副本

### 8.1 目标状态

Node4 加入后承担：

- 1 个 Sub2API 副本，将集群的 Sub2API 副本数从 3 调整为 4；
- 1 个本机 Caddy 实例，固定代理 Node4 上的 Sub2API；
- 不承载 PostgreSQL 或 Redis；
- 与其他业务节点相同，Sub2API 每节点最多 1 个副本。

Node4 默认加入为 Worker。若按当前建议将 Node1–Node3 组成 3 Manager 控制面，其 quorum 为 2，可容忍 1 个 Manager 故障；将 Node4 也提升为 Manager 后，4 Manager 的 quorum 为 3，仍只能容忍 1 个故障，因此没有提升容错能力。Node1–Node3 的最终角色仍以实施时的 Swarm 评审结果为准；Node4 不因增加业务副本而自动获得 Manager 角色。

### 8.2 新节点前置检查

本地验证环境建议沿用 Node1–Node3 基线：Ubuntu 24.04、`linux/arm64`、2 vCPU、4G 内存、20G 磁盘。生产环境应使用 `linux/amd64`，单节点建议至少 16G 内存、200M 带宽，并按容量测试结果调整。

加入前检查：

- 主机名唯一且计划使用 Swarm 节点名 `node4`；
- 系统时间同步，Docker Engine 版本与现有集群兼容；
- 节点间开放 `2377/TCP`、`7946/TCP+UDP`、`4789/UDP`；
- 生产公网入口开放 `80/TCP`、`443/TCP`，端口没有被其他进程占用；
- 可以拉取私有 GHCR 镜像，并能访问共享 PostgreSQL、Redis；
- host-network Caddy 使用的 Redis storage 地址从 Node4 可达；
- 本地使用文档定义的缩小资源档；生产 Sub2API 必须设置硬内存上限，Caddy 至少预留 1G 内存；
- 不给 Node4 添加 PostgreSQL/Redis placement label。

### 8.3 创建本地 Multipass 节点

本地测试可先创建虚拟机：

```bash
multipass launch 24.04 \
  --name node4 \
  --cpus 2 \
  --memory 4G \
  --disk 20G

multipass info node4
multipass shell node4
```

记录 `multipass info node4` 返回的实际 IP，不在配置中假定固定地址。生产服务器不执行这一步。

### 8.4 安装 Docker 并加入 Swarm

在 Node4 上按项目批准的方式安装 Docker Engine。GoTask 只需安装在指定 Manager 或 CI/CD runner 上，无需安装到 Node4。

先在 Manager 获取 Worker 加入命令：

```bash
docker swarm join-token worker
```

然后在 Node4 执行命令输出的等价内容：

```bash
docker swarm join \
  --token <worker-token> \
  <manager-private-ip>:2377
```

Worker token 属于敏感信息，不得提交到 Git、写入 Taskfile 或复制到发布摘要。加入后在 Manager 确认节点身份，并立即转为 `drain`，避免校验完成前调度业务：

```bash
docker node ls
docker node update --availability drain node4
task ops:node-status NODE=node4
```

### 8.5 配置 Node4 并增加副本数

确认 Node4 的架构、网络、磁盘、镜像仓库认证和共享依赖后，在 Manager 添加业务 label：

```bash
docker node update \
  --label-add sub2api=true \
  --label-add caddy=true \
  node4
```

不要添加 PostgreSQL/Redis label。随后把环境清单中的业务节点集合（以及生产档的 `cluster_nodes`）从 3 个节点调整为 4 个。当前方案已固定使用 constrained `global` service，不配置 `replicas`：`sub2api=true` 和 `caddy=true` label 会让 Node4 自动各获得 1 个 task，也就是新增第 4 个 Sub2API 副本和第 4 个 Caddy 实例。

本地环境清单应增加 Node4 的实际地址和角色，例如：

```yaml
nodes:
  - {name: node4, address: <node4-ip>, roles: [caddy, sub2api]}
```

生产环境则将 `cluster_nodes` 调整为 `4`，并在受审核的节点清单中增加同样的 `caddy`、`sub2api` 角色；不得为 Node4 增加数据服务角色。

先生成变更计划，确认输出只包含 Node4 接纳、节点清单和 `global` task 数量变化：

```bash
task validate:environment ENV=local
task validate:stack ENV=local
task release:plan ENV=local RELEASE=v0.1.165-ext.1
```

通过审核后让 Node4 恢复调度；Swarm 会根据现有 `global` service 自动创建两个 task，无需为了扩容重新构建镜像：

```bash
task ops:undrain NODE=node4 CONFIRM=undrain-node4
task release:verify ENV=local
task ops:node-status NODE=node4
task ops:status
```

新增节点本身不要求生成新的应用版本；Node4 自动拉取当前 service 已固定的 digest。节点清单变更必须进入 Git 审核，实际 task 的 image digest 必须与当前 release 一致。

### 8.6 验证 Node4

Node4 至少需要通过：

1. Caddy 与 Sub2API 各有且只有 1 个 task，并且运行在 Node4；
2. Sub2API 使用当前环境对应架构的固定 digest；
3. Sub2API 能连接共享 PostgreSQL、Redis，`/ready` 返回成功；
4. Caddy 已加载所需模块并连接共享 Redis storage；
5. Node4 能获得与其他节点一致的证书，HTTPS 请求可达本机 Sub2API；
6. Node4 不存在 PostgreSQL/Redis task；
7. 容器资源 reservation/limit 与该环境的资源档一致。

本地可用 Node4 实际 IP 验证：

```bash
curl --noproxy '*' \
  --resolve sub2api.test:443:<node4-ip> \
  --cacert <caddy-local-ca.pem> \
  https://sub2api.test/ready
```

生产环境只有在上述验证完成后，才人工向 DNSPod 添加 Node4 的 A 记录并观察实际流量、带宽、内存和错误率。第一期不由 GoTask 修改 DNS，也不自动故障摘除。

### 8.7 移除 Node4

生产环境先人工移除 Node4 的 DNS 记录，等待 TTL 和已有连接消退，然后在 Manager 执行：

```bash
task ops:drain NODE=node4 CONFIRM=drain-node4
task ops:node-status NODE=node4
task ops:status
```

确认 Node4 已无 task 后，移除业务 label；再在 Node4 执行 `docker swarm leave`，最后由 Manager 移除节点对象：

```bash
docker node update \
  --label-rm sub2api \
  --label-rm caddy \
  node4

# 在 Node4 执行
docker swarm leave

# 回到 Manager 执行
docker node rm node4
```

同时将发布清单的业务节点数恢复为 3，并经 `plan -> apply -> verify` 固化期望状态。除非节点已经永久失联且确认没有残留 task，否则不使用 `--force`。删除 Multipass 虚拟机属于独立的破坏性操作，不纳入通用 GoTask 运维任务。

## 9. 故障处理

先收集状态，不因单个错误直接重启或改配置：

```bash
task ops:status
task ops:logs SERVICE=sub2api
task ops:logs SERVICE=caddy
task ops:node-status NODE=node4
```

| 现象 | 处理原则 |
| --- | --- |
| `validate:*` 失败 | 停止部署，修正 context、变量、label、架构、digest 或依赖，不使用 `--force` 绕过 |
| Node4 拉取镜像失败 | 核对 registry 认证、平台 digest 和 `--with-registry-auth`，不改用可变 tag |
| `/ready` 失败 | 暂停滚动更新，结合 Sub2API/Caddy 日志及 PostgreSQL/Redis 连通性定位 |
| 仅部分副本更新成功 | 保留现场证据；无法在维护窗口内修复时，回滚到有完整记录的上一 release |
| PostgreSQL/Redis 不可用 | 不临时迁移到空 volume；保持 readiness 失败并恢复共享依赖 |
| Node4 TLS 异常 | 核对 Caddy 模块、共享 Redis storage 配置、Secret、节点时钟和网络 |
| Node4 验证失败 | 保持 `drain` 或重新 `drain`，不加入 DNS，修复后再次完整验证 |

## 10. 操作检查清单

### 10.1 发布前

- [ ] 当前终端连接的是目标 Swarm 和指定 Manager；
- [ ] 没有另一个操作者或 CI Job 同时发布；
- [ ] `ENV`、`RELEASE` 与目标环境一致；
- [ ] 所有应用镜像均为平台匹配的固定 digest；
- [ ] Config/Secret 版本与 release 匹配且无明文 Secret；
- [ ] Stack 未隐式修改 PostgreSQL、Redis、Caddy 或数据卷；
- [ ] 更新策略、失败暂停和回滚目标已确认；
- [ ] 涉及数据服务时，备份与恢复路径已确认。

### 10.2 发布后

- [ ] 所有目标节点各运行最多 1 个 Sub2API 副本；
- [ ] 每个节点的本机 Caddy `/ready` 验证成功；
- [ ] 实际 image digest、Config/Secret、placement 与 plan 一致；
- [ ] PostgreSQL/Redis 连接正常，无 migration 并发异常；
- [ ] 日志、错误率、内存、重启次数和带宽无异常；
- [ ] 发布记录保存版本、digest、配置版本、时间、操作者和验证结果。

### 10.3 Node4 接纳

- [ ] Node4 以 Worker 加入，主机名、架构和网络检查通过；
- [ ] 接纳期间保持 `drain`，未添加 PostgreSQL/Redis label；
- [ ] 已将 Sub2API 目标副本数由 3 调整为 4；
- [ ] Node4 上只有 1 个 Sub2API 和 1 个 Caddy task；
- [ ] Node4 的资源限制、共享存储、TLS 和 `/ready` 验证通过；
- [ ] 生产 DNS 只在验证完成后人工增加；
- [ ] 回退时可移除 DNS、drain Node4，并恢复为 3 个副本。

## 11. 参考资料

项目内文档：

- [`Sub2API-MultiNode-Deployment.md`](./Sub2API-MultiNode-Deployment.md)
- [`Multipass-Nodes.md`](./Multipass-Nodes.md)

上游与官方文档：

- [wuhanstudio/app-docker-swarm](https://github.com/wuhanstudio/app-docker-swarm)
- [GoTask Installation](https://taskfile.dev/docs/installation)
- [GoTask Guide](https://taskfile.dev/docs/guide)
- [GoTask Taskfile Schema](https://taskfile.dev/docs/reference/schema)
- [Docker stack deploy](https://docs.docker.com/reference/cli/docker/stack/deploy/)
- [Docker Swarm rolling updates](https://docs.docker.com/engine/swarm/swarm-tutorial/rolling-update/)
- [Docker Swarm add nodes](https://docs.docker.com/engine/swarm/swarm-tutorial/add-nodes/)
- [Docker Swarm drain a node](https://docs.docker.com/engine/swarm/swarm-tutorial/drain-node/)
