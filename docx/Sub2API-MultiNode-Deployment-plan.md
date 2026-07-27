# Sub2API 多节点部署实施计划

> 状态：`G0/G1/G2/G3` 已通过；`G4-A` 三副本启用已完成，`G4-B` 故障演练未授权
> 创建日期：2026-07-26
> 适用范围：三个 Multipass ARM64 节点的本地 Docker Swarm 验证，以及 AMD64 生产制品与配置基线
> 方案来源：[`Sub2API-MultiNode-Deployment.md`](./Sub2API-MultiNode-Deployment.md)
> 运维契约：[`GoTask-runbook.md`](./GoTask-runbook.md)
> 节点事实：[`Multipass-Nodes.md`](./Multipass-Nodes.md)

## 1. 目的与当前边界

本文把已经完成的多节点部署方案拆解为可执行、可验证、可停止和可回滚的实施步骤。阶段编号严格沿用方案文档第 7 节：阶段 0 至阶段 5。

当前已完成 `G1` 仓库侧实施、`G2` 首版制品发布、`G3` 本地单副本基线、阶段 3 多实例前置收敛和 `G4-A` 三副本启用。后续仍不执行下列操作：

- 不继续扩大阶段 3 运行时代码范围；新增修补仍须先有失败证据并重新审核白名单；
- 不执行 `G4-B` 的 task kill、节点/依赖中断、OOM、受控 migration 失败、失败暂停或实际回滚演练；
- 不执行生产部署、真实数据迁移或切流；
- 不配置 DNSPod，不处理 DNS 故障节点摘除；
- 不重复触发已完成的发布 workflow，不覆盖任何 release tag 或镜像 tag。

批准本文只代表认可实施顺序和门槛，不自动授权任何源码、外部制品、节点或生产变更。

## 2. 实施原则

1. **最小改造**：先使用共享 PostgreSQL/Redis、Swarm、Caddy、Secret/Config 和资源限制解决问题；只有无法由部署消除的多实例安全风险才进入 `backend/extends`。
2. **新增文件优先**：ext 实现及其实现测试优先新增到 `backend/extends`；原项目已有文件只保留必要的 Wire/router/参数等薄接入点，并逐项登记原因和修改范围。涉及原包私有行为的回归测试允许就地新增，不为目录合规导出 API 或增加包装层。
3. **若无必要勿增实体**：默认不新增 Ent/domain 实体、数据库表、leader 服务、配置中心、通用插件机制或调度框架。
4. **扩展默认启用**：确认需要的 ext 修补不设置功能开关；三个副本运行相同版本和行为。
5. **部署与代码分离**：`backend/extends` 只存代码修补；`deploy/cluster` 只存集群配置、GoTask 入口和必要短脚本。
6. **不可变发布**：镜像、Config、Secret 均不原地覆盖；生产部署使用平台镜像 digest，本地归档部署同时固定 source image ID、归档 SHA-256 与 node image ID；应用更新只通过 Swarm 完成。
7. **本地先行**：先完成 ARM64 本地验证；AMD64 只形成制品与生产配置基线，不在本计划中执行生产部署或切流。
8. **证据先于结论**：每个阶段必须保存版本、digest、对象引用、节点/task 状态、日志和验收结果，不能只以 `/health=200` 或 `docker service ls` 判定成功。
9. **阶段门槛不可跳过**：前一阶段未通过或缺少对应授权时，不进入后一阶段。
10. **人工上游同步**：不设置自动同步频率；发生冲突时人工处理，不执行共享分支 rebase/force-push。

## 3. 当前实施基线

以下状态是制定本计划时的仓库快照，正式实施前必须重新核对：

| 项目 | 当前状态 |
| --- | --- |
| fork 分支 | `main`，跟踪 `origin/main` |
| G1 实施起点 commit | `74b34cec90ff0638b5d51eabbf962ff4002d0472` |
| G1 实施提交链 | `4077dd769f54e69cd8a6acec6b44ad5e322ba4d9`（静态骨架）→ `08825263b6b04e72e8bba45273d406969a900aac`（private GHCR 发布面收敛）→ `2842f9ba729dae6d6d7d58e1881a92730108286b`（关闭最终发布阻断）→ `5779d0b4b0d7b4821f2283afd667598380343386`（G1 文档闭环）；最终 CI `30206791653`、Security Scan `30206791734` 均通过 |
| upstream 基线 commit | `2730c1c43b29be003925b033f3f9e645e726bb8c` |
| upstream VERSION | `backend/cmd/server/VERSION = 0.1.165` |
| fork VERSION | `backend/extends/VERSION = ext.2`；独立递增，不随 upstream 重置 |
| 组合版本 | `0.1.165-ext.2`；annotated tag `v0.1.165-ext.2` 固定到 `9aca50a8fd1ad34de6ef6ecf08eb58800a19fa89`，tag object 为 `6651806af0e8bf5415d63c9be2f27e2839a7ffe0` |
| `backend/extends` | 已完成 Redis OAuth SessionStore 与最小 lifecycle manager；没有新增实体、功能开关或通用扩展框架 |
| `deploy/cluster` | 已创建通用 Stack、两套环境档、Caddyfile 和 GoTask 契约；本地候选清单已固定 `v0.1.165-ext.2` ARM64 归档身份 |
| release workflow | 唯一入口为 GHCR-only `workflow_dispatch`；只读组合双 VERSION并校验已有 tag，任何 digest push 前要求已有 package 为 private 或确认尚不存在，push 后再次确认 private 才提升三个不可变 tag；不创建 GitHub Release、不发布 Docker Hub、不发送通知 |
| GoReleaser | 两份兼容配置只保留本地制品构建及完整 fork `main.Version`、`Commit/Date/BuildType` 注入，不包含 `dockers`、`docker_manifests` 或其他 registry publisher；集群发布不调用 GoReleaser |
| G1 工具版本 | Go `1.26.5`、Docker Client `29.6.1`、GoTask `3.50.0`、GoReleaser `2.17.0`、actionlint `1.7.7` |
| 本地节点 | `node1`、`node2`、`node3`，均为 Ubuntu ARM64、2 vCPU、4G 内存、20G 磁盘 |
| Swarm/业务 service | 2026-07-27 已完成三 manager Swarm、共享数据 service、一次性 bootstrap 和三节点应用/入口扩容；Sub2API/Caddy 均为 `3/3`，PostgreSQL/Redis 均为 `1/1` |

若正式实施时 upstream VERSION、历史 ext 序号、节点状态或依赖版本已变化，先更新版本矩阵和实施输入，再继续；不得机械使用本快照。

## 4. 阶段依赖与授权门槛

```mermaid
flowchart LR
    G0["G0 计划审核"] --> S1["阶段 1 节点与基础设施基线"]
    S1 --> S2["阶段 2 数据服务与单副本基线"]
    S2 --> S3["阶段 3 多实例前置收敛"]
    S3 --> S4["阶段 4 三副本与故障演练"]
    S4 --> S5["阶段 5 环境交付"]
    S5 --> P["生产容量与可观测性补充方案"]
```

### 4.1 独立授权点

| 门槛 | 允许的动作 | 当前状态 |
| --- | --- | --- |
| `G0` 计划审核 | 只审核本文，不修改实施文件或环境 | 已通过 |
| `G1` 仓库实施授权 | 允许修改版本/发布文件，创建 `backend/extends`、`deploy/cluster` 和测试；不推送镜像、不修改节点 | 已通过 |
| `G2` 制品发布授权 | 允许向私有 GHCR 推送新的不可变 tag/manifest，并记录平台 digest | 已通过 |
| `G3` 本地环境实施授权 | 允许安装/配置 Docker、初始化 Swarm、创建 Secret/Config/service/volume，并在三个 Multipass 节点部署 | 已通过 |
| `G4-A` 三副本启用授权 | 允许滚动 node1 到已审核候选版本、分发固定本地镜像、给 node2/node3 添加应用/入口 label，并做非破坏性三节点验证 | 已完成 |
| `G4-B` 故障演练授权 | 允许在本地测试环境执行 task kill、节点停止、依赖中断、OOM、受控 migration 失败、失败暂停和实际回滚测试 | 未授权 |
| `G5` 交付确认 | 确认本地验收结论并关闭实施计划 | 未授权 |

任何授权都只覆盖表中动作。`G1` 不隐含 `G2/G3`，`G4-A` 不隐含 `G4-B`，本地完成不隐含生产授权。

### 4.2 总体阶段状态

| 阶段 | 状态 | 进入条件 | 退出条件 |
| --- | --- | --- | --- |
| 0. 需求冻结与架构决策 | 已完成 | 方案设计审核 | 本地设计项确认、实施计划形成 |
| 1. 节点与基础设施基线 | 已完成 | `G1`；涉及 GHCR/节点时再分别取得 `G2/G3` | 发布链、制品、配置骨架和三 manager 基线通过 |
| 2. 数据服务与单副本基线 | 已完成 | 阶段 1 通过且已取得 `G3` | PostgreSQL/Redis、单次 bootstrap、单副本与本机 Caddy 基线通过 |
| 3. 多实例前置收敛 | 已完成 | 阶段 2 通过且代码修补范围再次确认 | 必要 P0 修补、进程级测试、候选制品和三进程冷启动满足门槛；未启用 `node2`/`node3` 应用副本 |
| 4. 三副本与故障演练 | 进行中（`S4-A` 已完成） | 阶段 3 通过；三副本启用和故障演练分别取得 `G4-A`/`G4-B` | 三副本、TLS、滚动更新、回滚和故障矩阵通过 |
| 5. 环境交付 | 未开始 | 阶段 4 通过 | 交付物、限制和验收报告完成并取得 `G5` |

## 5. 阶段 0：需求冻结与架构决策

### 5.1 当前结论

阶段 0 已由方案文档完成。本文不重新讨论已确认架构，只在实施开始前做一致性复核。

### 5.2 实施前复核清单

- [x] 确认方案文档状态仍为“阶段 0 门槛已满足”；
- [x] 确认本计划已人工审核；
- [x] 确认三个现有节点均为 `manager + worker`，后续容量节点只作为 worker；
- [x] 确认 PostgreSQL 固定 `node1`、Redis 固定 `node2`，不做数据服务 HA；
- [x] 确认 Caddy 每节点只代理本机 Sub2API，不改用 Traefik/routing mesh；
- [x] 确认本地使用 `sub2api.test`、`tls internal`，不接 DNSPod；
- [x] 确认生产容量与监控细项继续延期，不作为本地阻塞项；
- [x] 明确 `G1/G2` 已完成，`G3` 已授权且不隐含 `G4`。

### 5.3 停止条件

- 方案与计划存在未解决冲突；
- 实施范围要求新增未批准实体、控制面或通用扩展框架；
- 用户仅批准文档而未批准对应实施门槛。

## 6. 阶段 1：节点与基础设施基线

阶段 1 分为仓库侧准备、制品发布和节点侧基线三部分。三部分分别受 `G1/G2/G3` 控制，不因同属一个阶段而合并授权。

### 6.1 S1-A：仓库与上游基线复核

- [x] 确认工作树范围，保留用户无关改动；不使用 `git add -A` 暗中纳入其他文件；
- [x] 核对 `origin`/`upstream` URL、当前分支和远端默认分支；
- [x] fetch `origin` 和 `upstream`，只读比较差异；未合并 upstream；
- [x] 读取当前 `backend/cmd/server/VERSION`，查询历史 fork tag，计算下一个未使用的 `ext.N`；
- [x] 记录 upstream commit、fork commit、Go/Docker/GoTask 版本和实施日期；
- [x] 运行实施前后端基线测试并区分既有失败；结果见 6.4.1。

产物：版本输入表、remote/branch 记录、基线测试记录。

### 6.2 S1-B：双 VERSION 与发布链最小改造

在 `G1` 授权后按独立提交实施：

- [x] 新增 `backend/extends/VERSION`，历史 tag 核验后首值为 `ext.1`；
- [x] 保持 `backend/cmd/server/VERSION` 不变；CI 在发布前后检查两个 VERSION 文件未变化；
- [x] 修改 `.github/workflows/release.yml`，只读组合 upstream VERSION 与 ext VERSION，并限制为手工触发的 GHCR-only 入口；
- [x] 校验格式分别为 `X.Y.Z` 与 `ext.N`，并校验触发 tag 严格等于 `v${FORK_VERSION}`；
- [x] 删除 workflow 中写入、上传替换或自动提交 `backend/cmd/server/VERSION` 的步骤；
- [x] 修改 `.goreleaser.yaml` 和 `.goreleaser.simple.yaml`，保留本地制品构建及完整 fork `main.Version` 与 `Commit/Date/BuildType` 注入，删除 `dockers`、`docker_manifests` 和其他 registry publisher；
- [x] release workflow 不创建 GitHub Release、不发布 Docker Hub、不更新 Docker Hub 描述、不发送 Telegram 通知；
- [x] 固定 release workflow 只生成 ARM64/AMD64 架构 tag 和 multi-arch tag，全部带完整版本且不可覆盖；
- [x] 任何 digest push 前要求已有 GHCR package 为 private 或确认尚不存在；两个架构按内容 digest 构建后再次确认 package 为 private，才提升最终 tag；部分架构 tag 只允许 digest 完全一致时恢复，禁止覆盖；
- [x] 增加只读校验：两个 VERSION 文件未被 CI 修改、tag/运行时版本一致、历史 ext 序号未复用。

验证：

- [x] `git diff` 证明 upstream VERSION 未变化；
- [x] 两份 GoReleaser 兼容配置由固定 GoReleaser `2.17.0` 校验，不使用浮动 GoReleaser 版本执行集群发布；
- [x] 本地 dry-run 得到 `FORK_VERSION=0.1.165-ext.1`；
- [x] 使用 Go `1.26.5` 构建的二进制返回完整 fork `Version/Commit/Date`；
- [x] 没有为 `-ext.N` 修改应用内更新逻辑。

### 6.3 S1-C：Caddy 与双架构制品输入

- [x] 使用固定的 Caddy `v2.11.4` 和 `github.com/pberkel/caddy-storage-redis@v1.8.1`，并核对 tag 对应源码 commit；
- [x] 以单一最小 Dockerfile 作为 Caddy 自定义镜像构建输入，不建立额外镜像框架；
- [x] Caddy 使用独立手工构建 workflow/package，不进入 Sub2API 镜像；
- [x] 分别构建 `linux/arm64`、`linux/amd64` 子镜像和 multi-arch manifest；
- [x] Sub2API 使用现有多阶段 Dockerfile 和 Buildx 的 digest-first 双架构发布链，不复制应用构建逻辑；
- [x] PostgreSQL `18-alpine`、Redis `8-alpine` 固定同一上游 index 下对应平台 digest；
- [x] 保存构建输入、源码 revision、G1 模块清单和平台 digest；现有链路未直接产出 SBOM/扫描，未为 G2 新增扫描工具。

只有取得 `G2` 后才允许推送以下私有 package：

- `ghcr.io/ryanpenn/sub2api`
- `ghcr.io/ryanpenn/sub2api-caddy`

验证：

- [x] 发布 workflow 的 `docker buildx imagetools inspect` 证据显示 ARM64/AMD64 平台正确；
- [x] G1 本地源码构建的 `caddy version` 为 `v2.11.4`；G3 本地归档镜像已再次复验；
- [x] G1 本地源码构建的 `caddy list-modules` 包含 `caddy.storage.redis`；G3 本地归档镜像已再次复验；
- [x] 部署记录使用平台子镜像 digest，而不是只记录 manifest tag；
- [x] 生产 GHCR 不可达时停止，不回退到 Docker Hub、`latest` 或未核验制品；Multipass 本地只使用三重校验的固定输入归档。

### 6.4 S1-D：`deploy/cluster` 最小骨架

在 `G1` 授权后创建下列最小结构；不创建空控制器、占位框架或长期运行组件：

```text
deploy/cluster/
├── Taskfile.yml
├── promote-ghcr-manifests.sh
├── taskfiles/
│   ├── validate.yml
│   ├── release.yml
│   └── ops.yml
├── stacks/
└── env/
    ├── local-arm64/
    └── production-amd64/
```

- [x] 根 Taskfile 只组合 `validate/release/ops` 命名空间；
- [x] `promote-ghcr-manifests.sh` 仅校验并提升 GHCR 架构 digest/manifest，供两份手工 Workflow 复用，不承担构建、授权或部署；
- [x] `validate` 覆盖 Docker Context、Manager/quorum、架构、label、placement、资源限制、固定 digest 和 Config/Secret 引用；
- [x] `release` 实现 `plan/apply/verify/rollback/bootstrap` 契约，不提供通用 `uninstall`；
- [x] `ops` 第一期只提供状态、日志和节点检查；未增加 `drain/undrain` 自动化；
- [x] ARM64/AMD64 环境文件只保存非敏感参数和脱敏模板；
- [x] Stack 不包含明文 Secret、可变 tag、Docker Socket 或未受控 bind mount；
- [x] 两套 Caddyfile 包含本机 upstream、Redis storage、回环 admin API、入口 TLS 和管理端更新接口阻断规则；集群内 Redis 首期不虚构未实现的传输层 TLS；
- [x] 未创建空 `scripts/`、占位控制器或长期 bootstrap/migration service。

阶段 1 只要求配置可渲染、可静态校验；没有 `G3` 时不得执行 `release:apply`。

#### 6.4.1 G1 验证记录

- G1 完整实施提交链为 `4077dd769f54e69cd8a6acec6b44ad5e322ba4d9`（版本、Caddy 和集群静态骨架）→ `08825263b6b04e72e8bba45273d406969a900aac`（private GHCR 发布面收敛）→ `2842f9ba729dae6d6d7d58e1881a92730108286b`（移除 GoReleaser 第二发布入口并前移 private 校验）→ `5779d0b4b0d7b4821f2283afd667598380343386`（G1 文档闭环）；首个提交未按原计划拆分，但不重写已推送历史，审核修正均保持独立提交；最终 CI `30206791653` 与 Security Scan `30206791734` 均通过；
- G1 提交审核确认 `backend/extends` 未增加运行时代码或实体，`deploy/cluster` 与业务修补隔离；两轮审核发现的越权发布面、可变 tag、半完成发布恢复、GoReleaser 第二发布入口和 private 校验顺序问题均已按授权收敛，尚未运行发布 workflow；
- 发布面收敛静态验证：固定 `actionlint 1.7.7` 并调用 ShellCheck 检查两份 workflow 通过；固定 GoReleaser `2.17.0 check` 检查两份仅本地制品配置通过；提升脚本通过 `bash -n`、ShellCheck 以及“全新提升/相同 digest 部分恢复”两条无 registry 写入的 mock 回归；

- `actionlint 1.7.7`：两份 GitHub Actions workflow 通过；
- GoReleaser `2.17.0 check`：两份仅本地制品配置通过，且均不包含 registry publisher；
- Go `1.26.5`：`go test ./...` 全量通过；组合版本二进制输出 `0.1.165-ext.1`、起点 commit 与构建时间；
- Caddy：固定源码 revision 与 module revision 已由 Go module origin 核对；本地源码构建成功，版本和 module 清单通过，两套 Caddyfile `adapt` 通过；
- Stack/Taskfile：G1 时两套环境均完成 `docker stack config` 渲染，`task --list-all` 只暴露已批准入口；当时 `validate:stack` 按设计拒绝全零 digest，G2 已完成回填；
- 分阶段探针：阶段 2 的本地档使用现有 `/health`，阶段 3 readiness 修补验证后切换 `/ready`；生产档强制 `/ready`，避免在尚无该路由时阻断单副本基线；
- 前端：pnpm `9.15.9` 的 typecheck 和 production build 通过；全量 Vitest 为 `188 passed / 1 failed` 文件、`1301 passed / 2 failed` 用例，并有 10 个既有 mock 未处理错误。失败集中在 rollback API 第三个 timeout 参数断言及 `getLiveCapability` mock 缺失，本轮未修改前端源码；
- G1 时 Docker BuildKit 曾因不可达代理 `127.0.0.1:7890` 阻断 Caddyfile 构建检查；G3 清理 Docker Desktop 失效代理并重启后，本地 ARM64 Sub2API/Caddy 镜像均构建成功，Caddy 容器内 version/module 复验通过。G2 的双架构 manifest/digest 记录继续作为生产基线。

#### 6.4.2 G2 制品发布记录

- annotated tag `v0.1.165-ext.1` 的 tag object 为 `b5ca76496d45d921db56895816311113bba94cae`，peel 后固定到 G1 闭环提交 `5779d0b4b0d7b4821f2283afd667598380343386`；未创建 GitHub Release；
- Sub2API workflow [`30207208963`](https://github.com/ryanpenn/sub2api/actions/runs/30207208963) 最终 attempt 2 成功：首次架构 push 后 GitHub 将新 package 初始化为 public，post-push private gate 在提升最终 tag 前停止；人工将精确 package 改为 private 后，仅重跑失败的 publish job，并复用相同架构 digest；
- Caddy workflow [`30207210054`](https://github.com/ryanpenn/sub2api/actions/runs/30207210054) 成功：amd64 push 创建 package 后，在 arm64 构建完成及 publish 前将精确 package 改为 private，private gate 与不可变提升均通过；
- 两个 package 当前均为 private；发布只产生带完整版本的 ARM64/AMD64 架构 tag 和 multi-arch tag，没有 Docker Hub、`latest`、GitHub Release 或外部通知写入；
- G2 未进入任何 Multipass 节点；G3 没有扩大本地 token scope，而是使用校验归档把本地构建镜像分发到 node1，节点未配置 GHCR 凭据。

| 组件 | linux/arm64 digest | linux/amd64 digest | multi-arch/index digest |
| --- | --- | --- | --- |
| Sub2API `v0.1.165-ext.1` | `sha256:1845076d3ff9dd23c15e807c754438c2dc142d7b1ce8cdee2e407d903c543708` | `sha256:0186e45b9e2cf7a9dad65dadb0e342b9275764ddd3da406c48d343cd1e43e08f` | `sha256:dfff6a1333ebda168bbd0e868fba743c52f06c765aa3ae0beb373935a5e01f5f` |
| Caddy `v2.11.4-redis-v1.8.1` | `sha256:2e703acbd2db648195428f413f9338754481b6829752f330e35b5b901a01d531` | `sha256:b69f3df3fd10b6ec14db870047678e3be7cf511119169894100534404839cbed` | `sha256:c2ded406a07ebf438e0c0b7cbdd5a0773af6a78a65d8b6687c217a94934eb875` |
| PostgreSQL `18-alpine`（index `sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15`） | `sha256:122c9942437efcbbb8d595fc578dee7d26ee1543c2a8634d183adfa4a1e55b4d` | `sha256:b6a16ed0eb96e2c362811f7eeb951eac8b459e7b40be4149ea5444aa7c65569b` | 上游 index，仅部署平台子镜像 |
| Redis `8-alpine`（index `sha256:8096655e437712b07503796fb64d81359256cfcff0ab29d95a7da72863786efb`） | `sha256:ca5075df9552da2423c20c691a0208d60106f2ea71b47406d52c396bf0a6bd65` | `sha256:465aff338d817971674ff1ec3c0d59182e2b687018e87bf94b6e1491d0bb79e2` | 上游 index，仅部署平台子镜像 |

### 6.5 S1-E：节点与 Swarm 基线

取得 `G3` 后执行：

- [x] 重新核对三个节点的 IP、主机名、Ubuntu/ARM64、CPU、内存、磁盘和当前 service；
- [x] 核对时间同步、DNS、内核参数、端口占用、磁盘空间和 Docker 日志轮转；
- [x] 固定 Docker/GoTask 版本及安装来源，保存 checksum；
- [x] 由 `node1` 初始化 Swarm，`node2`/`node3` 以 manager 加入；三个 manager 均保留 worker 能力；
- [x] 验证 manager 数量为 3、三者 `Ready/Reachable` 且唯一 Leader；普通容量扩展不增加 manager；
- [x] 创建内部 overlay network；Caddy 继续使用 host network，不发布 routing mesh 入口；
- [x] 设置 `postgres=true` 仅在 `node1`、`redis=true` 仅在 `node2`；
- [x] `node2`/`node3` 已完成能力检查且暂不加应用入口 label；
- [x] `node1` 的 `sub2api=true`/`caddy=true` 在数据服务健康且一次性 bootstrap 成功后添加；阶段 4 再启用其余节点；
- [x] 指定 `node1` 为本地人工发布入口；本地应用/Caddy 镜像由开发机生成经 SHA-256 校验的归档并上传加载，不配置 GHCR 凭据；
- [x] 验证没有节点持久化 `write:packages` 或 `read:packages` 凭据；生产仍使用 registry/digest 交付。

### 6.6 阶段 1 退出门槛

- [x] 版本组合、tag、构建注入和双架构制品可追溯；本地构建的 Sub2API 版本输出与 Caddy version/module 已在 G3 复验；
- [x] ARM64 平台 digest 已回填本地环境；AMD64 digest 已记录但未部署；
- [x] `deploy/cluster` 静态渲染通过且未包含 Secret；生产模板仍由域名/IP/容量占位值阻断部署；
- [x] 三个 manager quorum 正常，数据节点 label 唯一；
- [x] 阶段 1 退出时尚未部署 PostgreSQL、Redis、Caddy 或 Sub2API service；随后已按 G3 进入阶段 2 数据服务部署；
- [x] 阶段 1 差异按版本发布、集群骨架、节点证据分开审核。

### 6.7 G3 节点实施记录（2026-07-27）

- 三个节点均安装 Docker Engine/CLI `29.6.1`，使用 Docker 官方 Ubuntu apt 仓库；GoTask `3.50.0` 仅安装在指定发布入口 `node1`；版本、仓库 key 与二进制 checksum 已登记到 [`Multipass-Nodes.md`](./Multipass-Nodes.md)；
- `node1` 为唯一 Leader，`node2`/`node3` 为 Reachable manager；三者同时保留 worker 能力。数据 label 为 `node1: postgres=true`、`node2: redis=true`；阶段 2 完成后仅给 `node1` 增加 `sub2api=true`/`caddy=true`；
- 已创建 attachable overlay network `sub2api-local-app`、两个内容寻址 Config 和五个版本化 Secret；一次性管理员密码 Secret 仅在 bootstrap 期间存在，成功后已删除；
- 运行态确认 OCI 镜像架构名 `arm64` 与 Docker 29.6.1 Swarm placement 字段 `aarch64` 不同，已分别使用 `TARGET_ARCH` 与 `SWARM_NODE_ARCH`，避免错误调度；
- 首次 Stack apply 在创建 service 前暴露 Compose 必填变量错误文本的插值歧义；已把 image 必填表达式收敛为无嵌套符号的 `${IMAGE:?required}`，并在静态校验中增加渲染后镜像精确相等检查；
- Redis 首次 task 因 Swarm 未按预期提供 service 级 `tmpfs` 目录而失败；已把启动时 ACL 临时文件收敛到容器 `/tmp`，再调用官方 entrypoint 完成数据目录权限修复及降权，Redis 主进程以 `redis` 用户运行；`node2` 已持久化 `vm.overcommit_memory=1`。应用 ACL `PING` 通过，Caddy ACL 仅允许约定前缀且拒绝前缀外读取；失败 task 作为历史证据保留，不阻断恢复后基于当前 desired state 的验收；
- PostgreSQL 已按 ARM64 固定 digest 运行于 `node1`，task 为 `1/1` 且 health 为 healthy，`pg_isready` 确认接受连接；service 无发布端口，主进程以 `postgres` 用户运行；
- 现有本地 token 与 keyring 凭据经 GHCR token/manifest 只读请求均返回 `403`。未扩大旧 token scope；改用开发机本地固定输入构建与归档上传，节点无需 GHCR 凭据；
- Sub2API 本地 tag 为 `sub2api-local/sub2api:v0.1.165-ext.1-arm64`，source image ID 为 `sha256:6114ce6ea99d734c40fdf7ccb1dbf7b88ea44785e44de99ea066f43a5a435fc0`，归档 SHA-256 为 `150e648aeefec2cd541807bb726e9ca4b4c243f4f1cf639045d50ce49a51da39`，node1 加载后 image ID 为 `sha256:658b62d53062a22140670a40622b65f69432c7f32293113e2960c74b826e1e04`；
- Caddy 本地 tag 为 `sub2api-local/caddy:v2.11.4-redis-v1.8.1-arm64`，source image ID 为 `sha256:401642d309bd0f0b8a7966aa72b12514f04166ca8c5cf8adefe2d4498824c4f0`，归档 SHA-256 为 `cc8f05e47661ca5b41998b884831abc8e126082cf9ed697cd82fdc56d9c92ff2`，node1 加载后 image ID 为 `sha256:26a85a756bcbd9d2f94d9bc55e48fce85ee55cf181b6002a3c82e1292504b739`；
- `task images:distribute-local ENV=local` 已验证开发机 image ID、平台、归档 SHA-256、node1 加载后 image ID 和远端临时文件清理；Stack 使用 `--resolve-image never`，生产档继续要求 registry digest；
- 一次性 bootstrap 首次完成 migration 和管理员创建后因临时数据目录缺失在配置落盘阶段退出；补充 `mkdir -p "$DATA_DIR"` 后幂等重跑成功，临时 service 与密码 Secret 均删除；
- Sub2API/Caddy/PostgreSQL/Redis 当前均为 `1/1`；HTTPS `/health` 返回 200，管理员登录成功。Caddy 强制重建后 Local CA SHA-256 指纹保持 `1C:F3:6C:A9:FF:B0:AE:B9:25:3E:B0:47:95:D4:76:5A:F0:41:B8:EE:3A:B7:7A:07:58:E4:F9:7A:89:93:A2:CB`，证明 Redis storage 可复用 CA；
- 管理员登录后的 `/api/v1/admin/system/version` 在首次合规确认前按上游逻辑返回 `423 ADMIN_COMPLIANCE_ACK_REQUIRED`；这是应用业务门槛，不是 TLS、认证或版本注入失败，首次登录页面确认后再复验完整版本；
- 本轮没有修改 `backend`，没有新增实体、控制面、第二套 Stack 或通用框架，也没有进入 G4 故障演练。

本次创建的非敏感对象引用如下；表中只记录名称与 Swarm object ID，不记录 Secret 内容或摘要：

| 类型 | 名称 | Object ID |
| --- | --- | --- |
| Config | `sub2api-local-caddyfile-e7861ad7e3f4` | `ynvvg8m2fjlgb12glq1jjc0ch` |
| Config | `sub2api-local-model-pricing-139de8a906ce` | `ceb9vk5ho18xufxcxufvqzo15` |
| Secret | `sub2api-local-app-config-v001` | `zigvsbccbvfjy72y9d9brm22f` |
| Secret | `sub2api-local-caddy-storage-key-v001` | `x6tykbq3757vol9sv4lg3w5pb` |
| Secret | `sub2api-local-postgres-password-v001` | `wzlyspnbsy57k3phhfqtdw8wm` |
| Secret | `sub2api-local-redis-app-password-v001` | `v4cc3hpdg9aoegc0eruvtrh0c` |
| Secret | `sub2api-local-redis-caddy-password-v001` | `nwyaoz706hl1i8pld3srp0n3g` |

## 7. 阶段 2：数据服务与单副本基线

### 7.1 S2-A：部署共享数据服务

- [x] 以固定 ARM64 digest 部署 PostgreSQL 单实例，placement 绑定 `node1` 和明确的本地持久化目录；
- [x] 以固定 ARM64 digest 部署 Redis 单实例，placement 绑定 `node2` 和明确的本地持久化目录；
- [x] 禁止两项 service 漂移到无原数据目录的节点；
- [x] PostgreSQL/Redis 不对测试入口公开，只通过内部网络或已确认私网端点访问；
- [x] Redis 启用本地 AOF/RDB 基础持久化；不把它表述为跨节点备份；
- [x] S3 配置保持为空，定时 S3 备份保持禁用；
- [x] 应用数据库账号、Redis ACL 和 Caddy TLS storage ACL 按消费者边界最小授权。

验证：健康检查、持久化目录、placement、重启后加载、网络暴露和资源限制均符合方案。

### 7.2 S2-B：创建 Config/Secret

- [x] Config 使用 `sub2api-{env}-{purpose}-{sha12}`；
- [x] Secret 使用 `sub2api-{env}-{purpose}-vNNN`，不记录 Secret 内容摘要；
- [x] 创建本地专用 `app-config`、PostgreSQL、Redis app、Redis Caddy、Caddy storage key 等 Secret；不复用生产值；
- [x] JWT/TOTP 默认收敛在 `app-config`，只有消费范围确实不同时才拆分；
- [x] 创建经审计的 `model_pricing.json` Config；本地第一期把 `pricing.remote_url`/`hash_url` 置空，仅使用该不可变 Config，避免为当前 fork 内快照额外建立远程镜像；生产准入时再固定与生产快照匹配的不可变 URL/hash；
- [x] 创建 Caddyfile Config；发布记录保存 Config 名称/内容摘要及 Secret 名称/object ID；
- [x] 长期 Secret 均通过受控输入创建，不进入 Git、Stack、Config、镜像或日志；一次性管理员密码按本次本地测试授权经 stdin 注入，成功后 Secret 已删除，未写入仓库；
- [x] 本地 Secret 丢失时按方案重建环境，不额外建设 Secret 备份系统。

### 7.3 S2-C：单次 bootstrap

- [x] 创建一次性 bootstrap 管理员密码 Secret；
- [x] 只启动一个临时受控实例，显式提供管理员密码、JWT/TOTP Secret 并设置 `AUTO_SETUP=true`；
- [x] 等待 migration、schema checksum、管理员和必要 seed 完成；
- [x] 验证 bootstrap 成功后关闭并删除临时实例；
- [x] 删除一次性管理员密码 Secret；
- [x] 正式 service 固定 `AUTO_SETUP=false`，只读挂载权威 `app-config`；
- [x] 不保留 bootstrap service，不新增 migration Job 或协调实体。

### 7.4 S2-D：单副本与本机 Caddy

- [x] 使用 `global` service，但阶段 2 只有 `node1` 带 `sub2api=true`/`caddy=true`，因此各运行一个 task；
- [x] Sub2API 以 host-mode `8080` 供本机 Caddy 使用；本地测试环境按本次授权接受该端口可从宿主机访问，生产准入前必须以防火墙或等价网络约束关闭绕过路径；
- [x] Caddy 使用 host network 绑定 `80/443`，admin API 仅监听 `127.0.0.1:2019`；
- [x] Caddy 固定代理本机 Sub2API，不通过 routing mesh；
- [x] Caddy 使用 Redis storage 的专用 ACL、DB、key prefix 和 encryption key；
- [x] 通过部署配置统一启用现有每副本图片 limiter，并为三个副本预设相同参数；当前仅作为 4G 本地资源基线，不解释为生产配额；
- [x] 本地使用 `sub2api.test` 与 `tls internal`；
- [x] 通过 `curl --noproxy '*' --resolve` 验证 TLS、`/health` 和管理员登录；`/ready` 留待阶段 3 修补后验证；
- [x] 将 SSE、WebSocket、生图和压力资源基线收敛到阶段 4 的多实例专项，不把 Provider 凭据或业务压测加入单副本部署门槛。

### 7.5 阶段 2 退出门槛

- [x] PostgreSQL 只在 `node1`、Redis 只在 `node2`，placement 禁止漂移到空目录；
- [x] bootstrap 只执行一次，一次性密码 Secret 已删除；
- [x] 正式实例 `AUTO_SETUP=false`，权威 Config/Secret 引用可追溯；
- [x] 单副本经本机 Caddy 正常访问；本地直连 `8080` 的安全例外已登记，不能外推到生产；
- [x] Caddy Redis storage 重启后仍能读取相同本地 CA/证书数据；
- [x] 4G 本地资源 reservation/limit 生效，但没有把结果解释为生产容量。

## 8. 阶段 3：多实例前置收敛

阶段 3 先用测试证明风险，再做最小修补。不得把方案中的“可能风险”直接转化为新功能。本阶段保持阶段 2 的单副本 Swarm 基线，不给 `node2`/`node3` 添加应用 label；并发行为通过目标单元测试、进程级集成测试和协议级 stub/mock 验证，真实三节点测试统一在阶段 4 执行。

### 8.1 S3-A：多实例风险与接入点盘点

- [x] 重新沿当前源码确认 OAuth SessionStore、并发槽清理、图片 limiter、readiness/drain、WebSocket、本地文件和后台任务调用链；
- [x] 为每项建立“风险证据 → 最小修补 → 测试 → 目录例外”映射；
- [x] 确认能由部署解决的项目不进入 `extends`；
- [x] 确认需要的既有目录 Wire/router/参数薄接入点，并在第 8.10 节登记实际文件；
- [x] 实施前按第 8.8 节白名单形成精确文件清单；基于失败证据新增的两个图片入口和 Scheduled Test 文件已同步收窄白名单；
- [x] 未新增实体、表或通用抽象。

### 8.2 S3-B：P0 代码修补

按独立、小范围提交实施，每项先补失败测试：

1. **并发槽清理**
   - [x] 先用原包回归测试复现：新副本启动会调用 cache-wide stale process cleanup；
   - [x] 最小修补仅删除 `backend/internal/service/wire.go` 中启动时调用 `CleanupStaleProcessSlots` 的路径；
   - [x] 新副本启动不再触发其他健康 request prefix 的清理；
   - [x] 继续复用现有 score/TTL/活跃索引和周期清理，不改写既有回收模型；
   - [x] 未创建 `extends/concurrency` 包装层，未新增 owner、heartbeat、接口或持久化模型；
   - [x] 已登记为“一处 upstream 原文件直接修补 + 原包 test-only 回归测试”例外。
2. **OAuth Redis SessionStore**
   - [x] `backend/extends/oauthsession` 只提供一个可复用的 typed Redis JSON store，统一 provider namespace、TTL 和错误语义；
   - [x] 一次性消费使用 Lua 在单次 Redis 操作中完成“读取、比较预期 state、匹配后删除并返回”；state 不匹配不删除，Redis 错误 fail closed；
   - [x] 五个 OAuth service 各自定义最窄 typed interface，只修改 store 字段、构造注入及 `Put/Take` 调用，不直接 import `extends`；
   - [x] `backend/extends/wire.go` 集中创建 typed store，composition root 接入一个 `extends.ProviderSet` 并以两个 `wire.Bind` 绑定 lifecycle 的 consumer interface；
   - [x] 五个 `internal/pkg/*/oauth.go` 上游内存实现保持不变；生产 Wire 只注入 Redis store，不依赖粘性会话、不回退进程内 store、不新增数据库实体。

### 8.3 S3-C：最小 readiness、排空与 WebSocket

- [x] 保留 `/health` 作为 liveness；新增 `/ready`，以 2 秒按需探测复用现有 PostgreSQL/Redis 客户端并表达 draining；
- [x] `SIGTERM` 先置 draining、拒绝新请求，再按 `server.shutdown_timeout=40` 秒排空；小于 Swarm `stop_grace_period=45s`；
- [x] HTTP/SSE 使用 `http.Server.Shutdown` 语义完成排空；
- [x] WebSocket 只增加最小客户端连接 registry：draining 后拒绝新 upgrade，已有连接可继续到窗口结束，到期并行发送 `1012 Service Restart` 后关闭；
- [x] 第一期不识别“当前 turn/新 turn”，未侵入 forwarding loop；精确 turn 语义继续作为 P1 延期；
- [x] 既有 `response_id -> conn_id`、`session -> conn_id` 和 turn state 保持原实现，未复制到 lifecycle/registry；
- [x] 未为 draining/连接 registry 新增 Redis key、数据库实体或跨副本协调状态。

### 8.4 S3-D：图片内存保护证据门槛

- [x] 本地与生产模板使用同一 `gateway.image_concurrency` 参数结构；未新增 ext 开关、Redis 集群总计数或 limiter 框架；
- [x] 同步 Responses、Images 和 Grok generation 已进入现有 limiter，未重复接入；
- [x] Batch worker 保持现有单 worker/副本和跨实例 job lock，未接入 HTTP limiter；
- [x] 测试证明 WebSocket 首个生图 turn 与 Gemini native `IMAGE` 响应路径绕过保护后，仅在这两个调用点接入同一个现有进程级 limiter；
- [x] 其他未启用或无高内存证据的路径未产生代码修改。

### 8.5 S3-E：后台任务与启动副作用

- [x] 已盘点实际启用的周期任务、claim 方式、外部副作用和并发语义；
- [x] Account expiry、Proxy expiry 保持既有条件更新/事务语义，不加锁、不改代码；
- [x] Scheduled Test 已确认多个进程会各自 `ListDue` 并执行外部测试，是本阶段唯一新增门控的任务；
- [x] Scheduled Test 只复用现有 Redis leader lock 与 PostgreSQL advisory fallback；
- [x] 生产注入 Redis/PostgreSQL 两个协调后端；两者均无法获取锁时跳过当次，不无锁执行；
- [x] 全新部署未配置 `backup_schedule`，`GetSchedule` 默认返回 disabled，S3 不启用定时备份；
- [x] 未创建 `extends/scheduler`、leader facade、任务注册表或通用调度框架。

### 8.6 S3-F：migration 与配置一致性测试

- [x] 通过 migration runner 的并发 session/锁重试测试确认同一 advisory lock 串行执行 migration SQL；本阶段未启用三个 Swarm 副本；
- [x] 获锁后重新核对 `schema_migrations`/checksum，不重复执行的测试通过；
- [x] 使用一个全新 PostgreSQL 数据库同时启动三个隔离容器：3.497 秒内全部 `/ready=200`；236 个 migration 文件恰好形成 236 条唯一记录，checksum 格式异常和重复 filename 均为 0；
- [x] 事务 migration 失败回滚、锁上下文超时和未完成初始化不进入 `/ready` 的单元/协议测试通过；
- [x] `*_notx.sql` 无效索引检查、受控清理、相同 checksum 重试和 execution mode 测试通过；未修改 migration 记录或业务数据；
- [x] 本阶段没有 schema/migration 变化，兼容性结论为 `backward-compatible`；
- [x] JWT bootstrap、Simple Mode 默认分组和安全 Secret 并发 bootstrap 的既有幂等/唯一约束测试随 `go test ./...` 通过；
- [x] 静态 Stack 只引用同一 `app_config` Secret source；object ID 与跨节点 JWT/TOTP 行为仍留到阶段 4 实机确认；
- [x] `task validate:stack ENV=local` 已核对模型价格 Config 名称/摘要；本地模板远程 URL/hash 为空，生产不可变 URL/hash 仍是生产准入项。

### 8.7 S3-G：部署层安全规则与进程级测试

- [x] 两套 Caddyfile 均阻断在线更新检查、可回滚版本查询、原地更新和原地回滚接口；
- [x] `/api/v1/admin/system/version` 保持可访问，并由现有 BuildInfo 返回完整 fork 版本；
- [x] 本地继续采用已审核的 host-mode `8080` 测试例外，未把它误判为已关闭；生产准入仍由防火墙或等价网络约束阻断绕过路径，不在阶段 3 增加网络控制面；
- [x] 完成目标单元测试、进程级并发测试及 OAuth/WebSocket 协议级 stub/mock；真实双/三副本测试仍留在阶段 4；
- [x] 沿用阶段 2 已验证的 Caddy shared Redis storage，并保留阶段 4 的跨节点证书协调/恢复测试步骤；
- [x] 阶段 3 退出时，`task validate:stack ENV=local` 与本地归档分发校验通过；候选镜像只加载 node1，正式 Swarm service 仍固定 `v0.1.165-ext.1`，未提前执行 `release:apply`；后续正式切换已在获授权的 G4-A 完成；
- [x] 人工上游同步演练不作为本地退出门槛；同步仍由人工按需发起。

### 8.8 修改隔离白名单

允许按需新增，未使用的目录或占位文件不得预建：

- `backend/extends/VERSION`；
- `backend/extends/oauthsession/`；
- `backend/extends/lifecycle/`，仅承载本节确认的最小 readiness/drain 行为；
- `backend/extends/wire.go`，仅在存在 ext provider 时创建；
- `deploy/cluster/` 中本阶段实际需要的配置、Taskfile 与短脚本。

允许薄修改或就地新增测试：

- `backend/cmd/server/wire.go` 与生成的 `wire_gen.go`：只接入一个 `extends.ProviderSet`；
- `backend/cmd/server/main.go`：只接入 draining 和 shutdown timeout；如确需配置时，`internal/config/config.go` 只增加一个运行时长参数，不增加功能开关；
- `backend/internal/server/http.go`、`router.go`、common route：只注入窄 readiness interface 并注册 `/ready`；
- `backend/internal/web/embed_on.go` 与既有测试：只把 `/ready` 加入嵌入式前端 bypass 列表，避免 readiness 被 SPA fallback 截获；
- `backend/internal/handler/wire.go`、`openai_gateway_handler.go` 与 `openai_live.go`：只注入窄 WebSocket lifecycle interface、登记两个既有 WebSocket 入口，并在已证明遗漏的 WebSocket 生图入口复用现有 limiter；
- `backend/internal/handler/gateway_handler.go` 与 `gemini_v1beta_handler.go`：只让 Gemini native 图片请求复用同一现有 limiter；该例外由失败证据触发，不推广为通用 limiter 框架；
- 五个 OAuth service 文件：只替换 store 字段、构造注入和 `Put/Take` 调用；
- `backend/internal/service/wire.go`：删除破坏性并发槽启动清理；仅当失败测试成立时，为 Scheduled Test 增加既有 leader-lock 注入；
- `backend/internal/service/scheduled_test_runner_service.go`：只在每轮 `ListDue` 前复用现有 singleton leader lock；
- 图片 handler：仅在测试证明遗漏的具体高内存入口增加现有 limiter 调用；
- 对应原包测试文件：允许验证私有行为，统一登记为 test-only 例外。

明确禁止：

- 修改 Ent schema、domain 实体或新增 migration；
- 新建通用 scheduler、plugin、leader facade、connection-state 或 limiter 框架；
- 新增跨节点 WebSocket 状态；
- 为把测试放进 `extends` 而导出私有 API、增加 wrapper 或重复 adapter；
- 修改五个 `internal/pkg/*/oauth.go` 内存 SessionStore，或为 fork 修改应用内更新逻辑。

### 8.9 阶段 3 验证与退出门槛

- [x] `gofmt`、目标单元/协议测试及 `cd backend && go test ./...` 通过；
- [x] Stack/Caddy/Taskfile 静态校验通过；
- [x] ext 实现及其实现测试位于 `backend/extends`；原包私有行为和薄接入点回归测试就地新增并已登记；
- [x] 实际修改文件全部位于第 8.8 节白名单，未为了目录合规导出 test-only API；五个既有构造器保留的内存 adapter 仅维持上游单元测试兼容，生产 Wire 不使用；
- [x] 未增加无关功能、功能开关、实体、表或通用框架；
- [x] 每项代码修补均有风险证据、修补测试和回归测试；
- [x] 没有失败证据的图片入口和后台任务没有产生代码修改；
- [x] `v0.1.165-ext.2`、fork commit、ARM64 source/node image ID、归档 SHA-256、既有 Config/Secret 和 `backward-compatible` schema 结论可追溯；AMD64 已完成同提交构建烟测但未推送 GHCR，不作为生产可部署 digest；
- [x] 已在第 8.10 节形成“不改代码基线”与“必要最小改造”的差异清单并完成运行态复核。

### 8.10 阶段 3 实施记录（2026-07-27）

当前状态：**阶段 3 已通过；其退出时的候选与隔离验证证据保持有效，后续 G4-A 已将该候选部署到三个正式副本。**

| 风险证据 | 必要最小改造 | 验证 | 未增加内容 |
| --- | --- | --- | --- |
| `ProvideConcurrencyService` 启动即调用 cache-wide stale cleanup | 删除这一处启动调用，保留既有周期/TTL 回收 | provider 回归测试确认启动调用次数为 0 | 无 `extends/concurrency`、owner、heartbeat |
| 五个 OAuth callback session 仅存进程内存 | 一个 typed Redis JSON store；五个 service 只持有各自最窄 interface | 跨 client 读取、TTL、namespace、state mismatch 保留、并发单次消费、Redis failure fail closed | 无粘性会话、DB 实体、生产内存 fallback |
| `/health` 无条件 200，退出只等待 5 秒 | `/ready` 按需探测 DB/Redis；进程内 draining；40 秒 shutdown | liveness/readiness 分离、依赖失败、draining、配置边界测试 | 无后台探针框架、功能开关 |
| 嵌入式前端 middleware 只 bypass `/health`，运行态把 `/ready` 截获成 SPA HTML 200 | 仅把 `/ready` 加入既有 bypass 列表 | embed 定向测试与候选容器返回 `{"status":"ready"}` | 无新 middleware、路由层或开关 |
| `http.Server.Shutdown` 不管理 hijacked WebSocket | 最小进程内 connection registry；两个 upgrade 入口薄登记；窗口到期并行关闭 1012 | 真实协议关闭状态测试 | 无跨节点连接状态、turn state、Redis key |
| WS 首个生图 turn 与 Gemini native `IMAGE` 绕过现有 limiter | 只在两个已证实入口复用同一个 `imageConcurrencyLimiter` | 拒绝第二个并发图片请求的目标测试 | 无新 limiter、集群总计数、Batch 改造 |
| Scheduled Test 每个进程独立 `ListDue` 并执行外部测试 | 每轮复用既有 Redis leader lock + PostgreSQL advisory fallback | peer 持锁时零 `ListDue`、单 leader 一次 `ListDue` | 无 scheduler/leader facade/任务注册表 |

验证记录：

- 阶段 3 源码提交链为 `8b1aec5c0af16979434aac4c121e5927dba3ee96`（多实例前置收敛）→ `9aca50a8fd1ad34de6ef6ecf08eb58800a19fa89`（运行态发现的 `/ready` SPA bypass 最小修补）；
- `cd backend && go test ./... -timeout=15m`：通过；
- `cd backend && go test -race ./extends/...` 及并发槽/Scheduled Test 定向 race：通过；
- OAuth、并发槽、Scheduled Test、readiness、WebSocket 1012、WS/Gemini limiter 目标测试：通过；
- embed 模式的 `/ready` bypass 定向回归：通过；首次候选冷启动发现 SPA 截获后以一行 bypass 修补闭环；
- `task validate:stack ENV=local` 与 `task images:distribute-local ENV=local`：通过；local health path 已切换为 `/ready`；
- annotated tag `v0.1.165-ext.2` 固定到 `9aca50a8fd1ad34de6ef6ecf08eb58800a19fa89`；ARM64 source image ID 为 `sha256:d6f956d592de70534e0c94fcff4199515dda555acc6f6ccef6405099daff5539`，归档 SHA-256 为 `3e1c69b1d96417acbd615ca7d48b8dbda60f070e65ccb6c0f80c59a095acae70`，node1 image ID 为 `sha256:bb638caa30eac89bf8bb5ee6395361f941f83fc0f810150249901ba896561703`；
- AMD64 同提交构建烟测通过，source image ID 为 `sha256:da61f82e6ec3dee3c0e9b311e224f970fa5f7245aa652cc7ddfa105f594da0a8`；未推送 GHCR，不写入生产部署清单；
- 全新数据库三进程并发 bootstrap 在 3.497 秒内全部 `/ready=200`；236/236 migration 唯一、0 checksum 异常、0 重复 filename，最终恰好 1 个管理员，另外两个实例幂等跳过；
- `git diff --check`：通过；
- 验证容器、临时数据库、Redis DB 15 和临时归档均已清理；阶段 3 退出时正式 Swarm service 仍为 `v0.1.165-ext.1`，当时未执行三副本、故障注入或生产发布；后续三副本启用单独记录在第 9.1 节。

阶段 3 的本地退出门槛已经关闭。host-mode `8080` 在本地仍是已接受的测试例外，生产防火墙验证属于生产准入而非本阶段伪完成项。阶段 3 通过本身不等于后续阶段授权；正式 service 的切换与三副本启用由后续获授权的 G4-A 单独完成。

## 9. 阶段 4：三副本与故障演练

阶段 4 拆分授权：`G4-A` 只覆盖三副本启用和非破坏性验证，已执行完成；`G4-B` 才覆盖故障注入、失败暂停和实际回滚，当前未授权。所有故障注入只允许作用于本地测试环境，不触碰生产或已有生产数据。

### 9.1 S4-A：启用三个 global 副本

- [x] 确认阶段 3 制品、Config/Secret 和回滚目标已固定；
- [x] 给 `node2`、`node3` 添加 `sub2api=true`/`caddy=true`；
- [x] 验证 Sub2API/Caddy `global` service 在每个合格节点恰好一个 task；
- [x] 验证每个 Caddy 固定代理本机 Sub2API，未经过 routing mesh；
- [x] 使用同一 Local CA，通过 `curl --noproxy '*' --resolve` 分别访问三个节点；
- [x] 核对三个 Caddy 的证书 subject、serial、指纹和 Caddyfile Config；
- [x] 核对三个 Sub2API 的固定 image ID、完整版本、配置对象和节点落位。

实施记录（2026-07-27）：

- 先通过 `release:plan -> apply -> verify` 将 node1 正式单副本从 `v0.1.165-ext.1` 更新为 `v0.1.165-ext.2`；运行镜像报告完整版本 `0.1.165-ext.2` 与 commit `9aca50a8fd1ad34de6ef6ecf08eb58800a19fa89`，Swarm `UpdateStatus=completed`，`PreviousSpec` 与 node1 保留的旧镜像均明确指向 `ext.1`，本轮未实际执行 rollback；
- 固定 ARM64 归档按既有 SHA-256 门禁装载到 node1/node2/node3，三个节点的 Sub2API image ID 均为 `sha256:bb638caa30eac89bf8bb5ee6395361f941f83fc0f810150249901ba896561703`，Caddy image ID 均为 `sha256:26a85a756bcbd9d2f94d9bc55e48fce85ee55cf181b6002a3c82e1292504b739`；
- node2、node3 依次添加 `sub2api=true`/`caddy=true`，每加入一个节点都等待 Sub2API/Caddy task Running 且该节点 `/ready=200` 后再继续；最终 Sub2API/Caddy 为 `3/3`，PostgreSQL/Redis 保持 `1/1`；
- 三个入口均通过同一 Caddy Local CA 根证书（SHA-256 指纹 `1C:F3:6C:A9:FF:B0:AE:B9:25:3E:B0:47:95:D4:76:5A:F0:41:B8:EE:3A:B7:7A:07:58:E4:F9:7A:89:93:A2:CB`）完成 TLS 与 JSON `/ready` 验证；叶证书 subject 为空、关键 SAN 均为 `DNS:sub2api.test`，serial 均为 `6A756405F963CC3B7D3310DCAF348F5B`，SHA-256 指纹均为 `40:FD:12:CF:C3:3C:C4:B8:45:80:75:AD:1F:09:91:C1:4E:A2:5D:FA:50:C5:F1:C2:3E:5C:C1:D5:A6:F3:15:7A`；
- 三个正式 task 共享同一个 app-config Secret object ID `zigvsbccbvfjy72y9d9brm22f`、模型价格 Config object ID `ceb9vk5ho18xufxcxufvqzo15` 和 Caddyfile Config object ID `ynvvg8m2fjlgb12glq1jjc0ch`；三个入口的在线更新检查均由 Caddy 返回 `403`；
- `/ready` 同时探测共享 PostgreSQL/Redis，三个节点均返回 200；近 10 分钟 Sub2API migration/panic/fatal 与 Caddy certificate/panic/fatal 关键错误计数均为 0。该结果只关闭 `S4-A`，不替代 OAuth、SSE/WebSocket、生图、续期/恢复、容量或故障专项。

### 9.2 S4-B：多实例功能专项

- [ ] 管理端、用户端和核心 API；
- [ ] HTTP、SSE、WebSocket 和最小滚动排空；精确 turn-aware 排空不作为第一期门槛；
- [ ] 所有 OAuth provider 的“节点 A 发起、节点 B 回调”、TTL 和一次性消费；无真实账号时用协议级 stub/mock；
- [ ] 对本环境实际启用且确认高内存的生图入口逐项验证 limiter；同步/异步复用路径不重复计数，Batch 继续验证既有 worker/job lock；
- [ ] 共享账户、Key、额度、调度、计费和模型价格；
- [ ] Scheduled Test 重复执行/锁回退/双协调后端不可用时跳过；Account/Proxy expiry 只验证既有安全并行语义；
- [ ] 三个真实 Swarm 副本同时启动下的 migration 并发、checksum、超时和恢复门槛；
- [ ] 三个正式副本引用同一 `app-config` Secret object ID，跨节点 JWT/TOTP 行为一致且日志不输出 Secret。

### 9.3 S4-C：滚动更新与回滚

- [ ] 通过 GoTask 执行 `release:plan -> apply -> verify`，逐节点验证 `/ready`；
- [ ] 更新策略为 `parallelism: 1`、`order: stop-first`、`failure_action: pause`；
- [ ] 验证更新节点本机入口存在预期短暂窗口，另外两个节点继续服务；
- [ ] 制造一个可恢复的 readiness/health 失败，确认滚动暂停而不是继续推进；
- [ ] 使用已记录的旧镜像 digest 与旧 Config/Secret 组合执行 rollback；
- [ ] 验证容器内二进制未被原地替换，三个副本最终版本一致；
- [ ] 单独滚动更新模型价格 Config，不重建镜像，验证旧/新短暂并存和旧价格回滚；
- [ ] 本地由当前执行者按手册完成并保存回滚证据；另一名执行者独立复现延期到生产准入。

### 9.4 S4-D：故障矩阵

按影响由小到大执行，每项先采集基线，失败后先保留证据：

| 场景 | 预期 |
| --- | --- |
| 停止单个 Sub2API task | 本节点 Caddy 短暂失败；其他节点继续服务；task 按策略恢复 |
| `SIGTERM`/滚动替换 | 新请求和新 WebSocket upgrade 被拒绝；已有 HTTP/SSE 按 shutdown 语义排空；已有 WebSocket 最迟在窗口到期返回 `1012`，不要求识别连接内新 turn |
| 单副本 OOM | 只影响该节点容量；记录 cgroup OOM、重启和 limiter 行为，不损坏共享状态 |
| 停止 `node3` | manager quorum 仍正常，容量下降一个副本；不在其他节点形成第二副本 |
| 停止 Redis | OAuth/共享缓存/Caddy storage 相关 readiness 符合预期；不误报健康；恢复后加载原数据 |
| 停止 PostgreSQL | 三个 Sub2API `/ready=503`；PostgreSQL 不漂移到空目录；恢复后重新挂载原目录 |
| 停止数据节点 | 控制面与数据服务故障边界符合方案；DNS 不自动摘除 |
| Caddy 重启 | 从共享 Redis storage 加载相同证书体系，不重复签发 |
| 管理端原地更新请求 | 由 Caddy 明确拒绝；`/version` 仍正常 |

禁止在本阶段执行：删除持久化卷、`docker stack rm` 通用卸载、破坏真实业务数据、生产 DNS 修改或未记录的 `--force` 删除。

### 9.5 S4-E：本地容量与稳定性记录

- [ ] 分别记录单/双/三副本普通请求、生图和长连接基线；
- [ ] 采集 `memory.current`、`memory.peak`、OOM、Go heap/GC、请求/响应字节、时长、网络和磁盘；
- [ ] 记录最热副本、DNS/入口分布偏差模拟和共享 PostgreSQL/Redis 瓶颈；
- [ ] 验证 4G 本地资源档的 reservation/limit 和有界拒绝语义；
- [ ] 明确报告本地数据不能推导生产配额或 200M 聚合带宽效果。

### 9.6 阶段 4 退出门槛

- [x] 三个 Sub2API/Caddy task 稳定且每节点最多一个；
- [ ] 多实例安全专项全部通过；
- [ ] shared TLS storage、续期协调和恢复行为通过；
- [ ] 滚动更新、失败暂停和旧组合回滚可复现；
- [ ] 故障矩阵均有带时间戳证据和明确结论；
- [ ] 未把本地验证表述为生产 HA、容量、DNS 摘除或灾难恢复证明。

## 10. 阶段 5：环境交付

### 10.1 交付物

- [ ] 更新后的最终部署清单与平台 digest 清单；
- [ ] `deploy/cluster` Stack、Caddy、环境模板、Taskfile 和必要脚本清单；
- [ ] GoTask 发布、验证、回滚、日志和节点命令清单；首期人工 drain/undrain 操作保留在手册，不建设自动化任务；
- [ ] Config 名称/内容摘要、Secret 名称/object ID 与镜像 digest 的发布对应表；
- [ ] 单次 bootstrap、migration、`*_notx.sql` 恢复和 forward-only 边界手册；
- [ ] Caddy TLS storage、本地 CA、证书协调、重启和恢复手册；
- [ ] ext 风险/修补/测试映射和原项目差异清单；
- [ ] fork/upstream 同步记录、版本组合和发布追溯记录；
- [ ] 本地多实例验收报告、故障矩阵、容量/热点/稳定性报告；
- [ ] 已知限制、遗留风险和 worker 扩容条件；
- [ ] DNSPod 多 A、公网 ACME、生产域名和切流的后续设计项；
- [ ] “当前不处理 DNS 故障节点摘除”的风险接受记录。

### 10.2 交付结论边界

本地交付可以声明：

- 三节点 Swarm 编排和多实例安全专项已按证据验证；
- ARM64 本地配置、双架构制品规则和 AMD64 配置基线已形成；
- 应用可通过增加合格 worker 和 label 横向增加 `global` task。

本地交付不得声明：

- 生产环境已部署或可直接切流；
- DNSPod 多 A 已验证均衡或自动故障摘除；
- PostgreSQL/Redis 具备 HA 或自动故障转移；
- 已满足后续 RPO/RTO；
- 4G ARM64 数据可以作为 16G AMD64 生产配额；
- 单个请求获得多节点带宽叠加。

### 10.3 生产前后续门槛

若进入生产，必须另行完成并审批：

- AMD64 单/三副本压测和当前生产峰值分析；
- Sub2API reservation/hard limit/`GOMEMLIMIT`、CPU、连接池、并发、队列、请求大小和服务目标；
- 生产指标后端、日志集中化、保留期、告警阈值、值班和升级流程；
- Secret 独立加密保管位置；
- forward-only migration 的备份恢复证据；
- DNSPod、公网 ACME、生产变更、切流和回退审批。

## 11. 提交与评审拆分

为便于同步 upstream 和回滚，实施提交不得把所有工作压成一个提交。建议按以下边界拆分：

1. `build: add fork version composition`
   - `backend/extends/VERSION`
   - release workflow 与两份 GoReleaser 配置
   - 版本组合/只读校验测试
2. `build: add pinned caddy image`
   - 单一 Caddy 构建输入与独立发布 job
   - 模块/平台验证
3. `deploy: add cluster validation skeleton`
   - `deploy/cluster` 最小目录、环境模板和静态校验
4. `deploy: add local single-node baseline`
   - PostgreSQL/Redis、bootstrap、单副本和 Caddy 配置
5. `fix: isolate concurrency slot cleanup`
   - 并发槽最小修补与测试
6. `fix: share oauth sessions through redis`
   - OAuth SessionStore 修补与测试
7. `fix: cover proven image limiter gap`（条件提交）
   - 仅包含失败测试证明的具体高内存入口；没有遗漏则不创建该提交
8. `fix: add readiness and graceful draining`
   - `/ready`、HTTP/SSE 和最小 WebSocket registry/到期 `1012`；不包含 turn-aware 框架
9. `fix: coordinate scheduled tests`（条件提交）
   - 仅在重复执行失败测试成立时，复用现有 leader lock；不包含 Account/Proxy expiry 或通用任务框架
10. `deploy: complete multinode acceptance flow`
    - Caddy 阻断、三副本发布/回滚、验收任务和记录模板

实际提交可进一步拆小，但不得把 upstream 同步提交与自定义修补混为一个不可审计提交。每次提交只 stage 明确文件，验证后再提交；是否 push 由当轮授权决定。

## 12. 跨阶段验证矩阵

| 类型 | 最低验证 |
| --- | --- |
| Go 代码 | 目标包测试、race-sensitive 测试、`cd backend && go test ./...` |
| 版本发布 | VERSION 只读、tag 关系、GoReleaser check、runtime `Version/Commit/Date` |
| 镜像 | 目标平台、固定 digest、Caddy module；现有链路可直接产出时附 SBOM/扫描记录 |
| Stack/Taskfile | YAML 解析、`docker stack config`、`task validate:*`、无明文 Secret/可变 tag |
| PostgreSQL/Redis | placement、持久化、health、ACL/连接、重启和非漂移 |
| Caddy | host network、本机 upstream、admin loopback、shared storage、TLS/更新接口阻断 |
| 应用 | HTTP/SSE/WebSocket、OAuth、limiter、readiness/drain、后台任务、seed |
| migration | advisory lock、checksum、三次冷启动、事务失败、notx 恢复、兼容性分类 |
| 发布 | plan/apply/verify/rollback、失败暂停、逐节点验证、旧组合恢复 |
| 资源 | cgroup peak/OOM、Go heap/GC、网络/磁盘、单/双/三副本对比 |
| 文档 | 本地引用、YAML 示例、实际命令、digest/对象引用和限制说明一致 |

## 13. 全局停止与升级条件

出现下列任一情况时停止当前阶段并请求人工决策：

- 工作树包含无法隔离的用户改动，或 upstream 基线已变化导致计划失效；
- 需要修改 `backend/cmd/server/VERSION`、覆盖已发布 tag/digest 或使用可变镜像；
- 修补需要新增未批准实体、表、控制面、通用框架或前端能力；
- ext 实现逻辑无法保持在 `backend/extends`，或原项目修改超出第 8.8 节白名单/不再是薄接入；原包 test-only 回归测试不触发此停止条件；
- GHCR、节点权限、manager quorum、平台架构或 digest 校验失败；
- Secret 可能进入 Git、命令行、日志、镜像层或发布摘要；
- PostgreSQL/Redis placement 可能漂移到空目录；
- migration 冷启动超过 5 分钟、checksum 异常、`*_notx.sql` 无恢复路径或出现未批准 forward-only 变化；
- 进程级或阶段 4 的双/三副本测试证明现有最小方案不能保证状态安全；
- 故障演练目标不明确、影响范围超出本地测试环境或缺少恢复路径；
- 任何步骤需要生产写入、DNS 修改、真实数据迁移或切流。

## 14. 计划审核清单

审核本文时重点确认：

- [x] 阶段顺序与方案文档第 7 节一致；
- [x] `G1` 至 `G5` 授权边界清楚；
- [x] 阶段 1 的版本/发布、制品、配置骨架和节点步骤已完成；
- [x] 阶段 2 单副本采用“仅 node1 添加应用/入口 label”的过渡方式已实施；
- [x] 阶段 3 按 P0/P1 和失败证据顺序实施；
- [x] 阶段 3 修改文件已同步收敛到第 8.8 节白名单，条件提交均有证据；
- [ ] `G4-B` 阶段 4 故障注入范围是否接受；
- [ ] 阶段 5 交付物是否足够；
- [x] `G1/G2/G3` 已分别授权并完成；`G4-A` 已完成，`G4-B` 未授权。

当前 G0/G1/G2/G3、实施阶段 0 至阶段 3 和 `S4-A` 均已通过。下一步先审核本节三副本启用证据，再单独决定是否授权 `G4-B`；未授权前不执行 task kill、节点/依赖中断、OOM、失败暂停、实际回滚或其他故障注入。
