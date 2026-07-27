# Sub2API 多节点部署实施计划

> 状态：本地实施基线已完成并通过 `G5` 交付确认；`G4-B2b-2b-2-fix/fix-deploy/retest` 与 `G4-B2c` 均已完成，当前三个 manager、Sub2API/Caddy `3/3`、PostgreSQL/Redis `1/1`，三个 HTTPS `/ready=200`，最终 `release:verify ENV=local` 通过。生产部署、DNSPod、真实数据迁移、容量定标及灾难恢复仍不在本次交付范围
> 创建日期：2026-07-26
> 适用范围：三个 Multipass ARM64 节点的本地 Docker Swarm 验证，以及 AMD64 生产制品与配置基线
> 方案来源：[`Sub2API-MultiNode-Deployment.md`](./Sub2API-MultiNode-Deployment.md)
> 运维契约：[`GoTask-runbook.md`](./GoTask-runbook.md)
> 节点事实：[`Multipass-Nodes.md`](./Multipass-Nodes.md)

## 1. 目的与当前边界

本文把已经完成的多节点部署方案拆解为可执行、可验证、可停止和可回滚的实施步骤。阶段编号严格沿用方案文档第 7 节：阶段 0 至阶段 5。

当前已完成 `G1` 至 `G5` 的本地实施基线。原 `G4-B2b-2a` PostgreSQL 暂停场景在 `ext.2` 下暴露 readiness 超时，已通过 `backend/extends/lifecycle` 两文件最小修补与 `ext.3` 复测关闭；原 node1/PostgreSQL 整节点场景又暴露正常退出 task 不会被 `on-failure` 重建，已通过仅修改部署层 Sub2API restart condition 为 `any`、既有静态断言、受控滚动和同场景复测关闭。单副本 OOM 与隔离 migration checksum 故障也已验证。后续仍不执行下列操作：

- 不继续扩大阶段 3 运行时代码范围；新增修补仍须先有失败证据并重新审核白名单；
- 不重复执行已经通过的故障注入；不把本地普通关机、短时依赖故障、单副本 OOM 或隔离 migration 失败外推为强制断电、磁盘损坏、跨节点/备份恢复、自动故障转移、Redis 持续不可用时 Caddy 冷启动或证书续期验证；
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
| fork VERSION | `backend/extends/VERSION = ext.3`；独立递增，不随 upstream 重置 |
| 组合版本 | 仓库与活动本地集群均为 `0.1.165-ext.3`；annotated tag `v0.1.165-ext.3` 已固定到版本提交 `6c859d2d83e03c49fb49a53e530932d7a6c789d7`，未上传 GHCR |
| `backend/extends` | 已完成 Redis OAuth SessionStore、最小 lifecycle manager，以及 PostgreSQL readiness 单 in-flight probe 与 caller 硬超时修补；没有新增实体、功能开关或通用扩展框架 |
| `deploy/cluster` | 已创建通用 Stack、两套环境档、Caddyfile 和 GoTask 契约；活动 `local-arm64/cluster.env` 已在提交 `3608d6c7b` 固定 `v0.1.165-ext.3` 的 source image ID、归档 SHA-256 与三节点 node image ID |
| release workflow | 唯一入口为 GHCR-only `workflow_dispatch`；只读组合双 VERSION并校验已有 tag，任何 digest push 前要求已有 package 为 private 或确认尚不存在，push 后再次确认 private 才提升三个不可变 tag；不创建 GitHub Release、不发布 Docker Hub、不发送通知 |
| GoReleaser | 两份兼容配置只保留本地制品构建及完整 fork `main.Version`、`Commit/Date/BuildType` 注入，不包含 `dockers`、`docker_manifests` 或其他 registry publisher；集群发布不调用 GoReleaser |
| G1 工具版本 | Go `1.26.5`、Docker Client `29.6.1`、GoTask `3.50.0`、GoReleaser `2.17.0`、actionlint `1.7.7` |
| 本地节点 | `node1`、`node2`、`node3`，均为 Ubuntu ARM64、2 vCPU、4G 内存、20G 磁盘 |
| Swarm/业务 service | G5 最终状态为三个 manager、Sub2API/Caddy `3/3` 与 PostgreSQL/Redis `1/1`；Sub2API condition 为 `any`、其他 service 为 `on-failure`，三个直连/HTTPS 入口和 `release:verify` 通过 |

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
| `G4-B1` 滚动与回滚授权 | 允许在本地测试环境执行 `S4-C` 受控滚动、失败暂停、旧清单回滚和模型价格 Config 回滚 | 已完成 |
| `G4-B2a` 低风险故障授权 | 允许在本地测试环境停止单个 Sub2API task、停止并恢复 node3、重启单个 Caddy task，并验证共享 TLS storage 读取 | 已完成 |
| `G4-B2b-1` Redis 依赖故障授权 | 允许在本地测试环境暂停并恢复同一 Redis 容器；不同时重启 Caddy、不停止数据节点 | 已完成 |
| `G4-B2b-2a` PostgreSQL 依赖故障授权 | 允许暂停并恢复同一 PostgreSQL 容器；不停止 node1、不修改 volume | 原 `ext.2` 执行未通过，`ext.3` 同场景复测已通过 |
| `G4-B2b-2a-fix` PostgreSQL readiness 最小修补授权 | 仅允许修改 `backend/extends/lifecycle/manager.go` 与 `manager_test.go`，实现单 in-flight probe、caller 硬超时并完成测试；不提升版本、不构建、不部署、不重复故障注入 | 已完成 |
| `G4-B2b-2a-candidate` 本地候选授权 | 审核修补提交；允许将 `backend/extends/VERSION` 提升到 `ext.3`，在本机构建并核验 ARM64 镜像；不创建 tag、不上传、不分发节点、不改活动清单、不部署 | 已完成 |
| `G4-B2b-2a-deploy-retest` 候选部署复测授权 | 允许分发已核验归档、切换本地活动清单、受控滚动三个副本并重复同一 PostgreSQL 容器暂停/恢复场景；失败时回滚 `ext.2`；不创建 tag、不上传、不执行其他故障 | 已完成并通过 |
| `G4-B2b-2a-tag` ext.3 标签闭环授权 | 允许创建 annotated tag `v0.1.165-ext.3` 并固定到版本提交 `6c859d2d83e03c49fb49a53e530932d7a6c789d7`，只推送 Git tag；不上传 GHCR、不修改运行态 | 已完成 |
| `G4-B2b-2b-review` 数据节点故障执行前审查 | 只读核对节点、quorum、placement、volume、数据不变量、入口预期、自动恢复与停止门槛；不停止节点或服务 | 已完成 |
| `G4-B2b-2b-1` node2/Redis 数据节点故障授权 | 允许普通停止并恢复 node2；禁止 `--force`、drain、改 label/service spec/volume 或同时停止其他节点 | 已完成并通过 |
| `G4-B2b-2b-2` node1/PostgreSQL 数据节点故障授权 | 仅在 `G4-B2b-2b-1` 完整恢复后，允许普通停止并恢复当前承载 PostgreSQL 的 node1；禁止 `--force`、drain、改 label/service spec/volume 或同时停止其他节点 | 已执行，未通过；禁止直接复测 |
| `G4-B2b-2b-2-recovery` 当前环境最小恢复授权 | 重新核对固定镜像、Config/Secret 与 rollout 门槛后，只允许对 `sub2api-local_sub2api` 执行一次受控 force-update，除 ForceUpdate generation 外不改变镜像、Config/Secret、placement、resource、update 或 restart 字段；恢复 node2/node3 task 并运行完整验证，不重部署 Stack、不触碰其他 service/节点 | 已完成并通过 |
| `G4-B2b-2b-2-fix-review` 配置层最小修正审查 | 只读复盘 Swarm heartbeat、healthcheck、restart policy 和 watchdog/验收时间关系，形成配置白名单与复测前门槛；不修改 Stack、源码或运行态，不执行故障 | 已完成并通过 |
| `G4-B2b-2b-2-fix` 静态配置修正授权 | 仅把 Sub2API `restart_policy.condition` 从 `on-failure` 改为 `any`，在既有 `validate:stack` 增加对应渲染断言并同步文档；不改其他 health/restart 参数、其他 service、源码或运行态 | 已完成并通过 |
| `G4-B2b-2b-2-fix-deploy` 配置应用授权 | 核对静态验证后，仅把已审核的 Sub2API restart condition 应用到本地 service，完成受控滚动和常态验证；不停止节点、不执行故障 | 已完成并通过 |
| `G4-B2b-2b-2-retest` node1/PostgreSQL 同场景复测授权 | 仅在配置应用与常态验证通过后，按修订后的 30/50/60 秒门槛重复同一普通节点停止/恢复场景；不执行其他故障 | 已完成并通过 |
| `G4-B2c` 资源与迁移故障授权 | 允许制造单副本 OOM 或受控 migration 失败 | 已完成并通过 |
| `G5` 交付确认 | 确认本地验收结论并关闭实施计划 | 已完成；仅确认本地实施基线 |

任何授权都只覆盖表中动作。`G1` 不隐含 `G2/G3`，`G4-A/G4-B1/G4-B2a/G4-B2b-1/G4-B2b-2a` 不隐含源码修补、`G4-B2b-2b-1/G4-B2b-2b-2/G4-B2c` 或生产授权；`G4-B2b-2a-fix` 不隐含候选发布，`G4-B2b-2a-candidate` 不隐含 tag、上传或部署，`G4-B2b-2a-deploy-retest` 不隐含 Git tag、GHCR 上传或其他故障注入，`G4-B2b-2a-tag` 也不隐含镜像发布或运行态变更。`G4-B2b-2b-review` 及 node1 执行前复审只关闭方案门槛；`G4-B2b-2b-1` 已独立授权并通过，`G4-B2b-2b-2` 已独立授权执行但未通过，`G4-B2b-2b-2-recovery` 只恢复环境且已通过，`G4-B2b-2b-2-fix-review` 只完成只读决策。只读审查不隐含 `G4-B2b-2b-2-fix`；静态修正不隐含运行态应用，配置应用不隐含节点故障复测，复测也不隐含 `G4-B2c`。已执行不等于已通过，仓库测试或本机构建通过也不等于故障门槛已通过。

### 4.2 总体阶段状态

| 阶段 | 状态 | 进入条件 | 退出条件 |
| --- | --- | --- | --- |
| 0. 需求冻结与架构决策 | 已完成 | 方案设计审核 | 本地设计项确认、实施计划形成 |
| 1. 节点与基础设施基线 | 已完成 | `G1`；涉及 GHCR/节点时再分别取得 `G2/G3` | 发布链、制品、配置骨架和三 manager 基线通过 |
| 2. 数据服务与单副本基线 | 已完成 | 阶段 1 通过且已取得 `G3` | PostgreSQL/Redis、单次 bootstrap、单副本与本机 Caddy 基线通过 |
| 3. 多实例前置收敛 | 已完成 | 阶段 2 通过且代码修补范围再次确认 | 必要 P0 修补、进程级测试、候选制品和三进程冷启动满足门槛；未启用 `node2`/`node3` 应用副本 |
| 4. 三副本与故障演练 | 已完成（本地授权范围） | 阶段 3 通过；三副本、滚动回滚和各故障子集分别取得对应独立授权 | 三副本、TLS、滚动更新、回滚和已授权故障矩阵通过；延期项登记为生产前门槛 |
| 5. 环境交付 | 已完成（本地实施基线） | 阶段 4 本地范围通过 | 交付物、限制和验收报告完成并取得 `G5` |

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

- [x] 以固定 ARM64 digest 部署 PostgreSQL 单实例，placement 绑定 `node1` 和 Stack-scoped Docker local named volume；
- [x] 以固定 ARM64 digest 部署 Redis 单实例，placement 绑定 `node2` 和 Stack-scoped Docker local named volume；
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

阶段 4 的本地授权项均已完成：三副本、受控滚动/暂停/回滚、单 task/单节点/Caddy、Redis 与 PostgreSQL 依赖、两个数据节点、单副本 OOM 和隔离 migration 失败均有现场证据。node1/PostgreSQL 的首次执行保持历史未通过，后续最小配置修正、运行态应用和复测另行记录为通过；任何结论都不外推到生产、真实数据、DNS 或灾难恢复。

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

- [x] 三个节点的管理端、用户认证、共享只读状态和核心 API 入口一致；当前无 Provider 账户，不把未发起的真实上游请求记为通过；
- [x] 普通 HTTP、管理 WebSocket 和 SSE/OpenAI WebSocket 协议级行为通过；最小滚动排空的实机部分保留到 `S4-C`，精确 turn-aware 排空不作为第一期门槛；
- [x] 所有 OAuth provider 的跨实例读取、TTL、state 校验和一次性消费已使用协议级 stub/mock 验证，符合无真实账号时的既定验收方式；
- [x] 已确认高内存的生图入口逐项通过现有 limiter 定向测试；同步/异步复用路径未重复计数，Batch 既有 worker/job lock 测试通过；
- [x] 共享用户与临时 Key 跨节点增删可见，模型价格一致；账户/额度/调度/计费在无 Provider 实体前提下使用既有定向与隔离集成测试验证；
- [x] Scheduled Test 单 leader、Redis lock、PostgreSQL fallback 代码路径及 Account/Proxy expiry 既有安全并行语义通过；双协调后端同时故障的实机跳过行为保留到 `G4-B2`；
- [x] 隔离 PostgreSQL integration harness 的 migration 并发串行、幂等、checksum 与 schema 最新性通过；三个正式 Swarm task 的受控同时重启未执行，保留到 `G4-B2`；
- [x] 三个正式副本引用同一 `app-config` Secret object ID，跨节点 JWT 签发、轮换和撤销一致，敏感日志扫描为 0；当前管理员未启用 TOTP，未虚构跨节点 TOTP 实机结果。

非破坏性实施记录（2026-07-27）：

- 在 node1 登录后，同一 access JWT 可在 node1/node2/node3 调用 `/api/v1/auth/me`；refresh token 在 node2 轮换后，新 access token 可在 node3 使用，旧 refresh token 被 node3 拒绝；随后在 node1 注销，新 refresh token 被 node2 拒绝。该流程验证共享认证状态，不在文档记录测试管理员密码或 token；
- 三个节点返回相同用户、API Key 列表、分组、`gpt-4o` 模型价格和完整版本 `0.1.165-ext.2`。为验证写后跨节点可见性，仅在 node1 创建一个额度为 1 的临时 API Key：node2/node3 均可见；从 node2 删除后 node1 返回 404、node3 列表恢复为 0，测试实体已清理；
- 三个 Caddy 入口的管理 QPS WebSocket 均完成 `101 Switching Protocols`。未携带 API Key 访问 `/v1/models`、`/v1/responses`、`/v1/messages` 时，三个入口返回一致的 401，证明请求进入本机应用入口；本轮没有 Provider 账户、Provider API Key 或 Scheduled Test plan，因此未制造真实模型调用、外部费用或额外业务实体；
- `go test ./extends/... -count=1`、handler/service/OAuth 定向测试和相关 `-race` 测试通过，覆盖跨实例 OAuth 一次性消费、readiness/WebSocket 生命周期、SSE 错误边界、图片 limiter、Batch job lock 与 Scheduled Test 单 leader；带 `unit` tag 的 Gemini native 图片和 Batch worker 测试通过；
- repository 隔离 integration harness 通过 migration 并发/幂等、Proxy expiry 与计费去重测试；正式共享数据库保持 236 条 migration、236 个唯一 filename、0 个空 checksum、0 组重复 filename，检查过程未修改 migration 或业务数据；
- 三个节点的 TOTP 状态均为 `enabled=false`；近 500 行正式 Sub2API 日志对 password、Bearer、refresh token、JWT/TOTP secret 等敏感模式扫描命中 0。正式 wiring 始终注入 Redis 与 PostgreSQL 两个协调后端，Redis error 会回退 PostgreSQL，数据库连接/查询失败返回未获锁并跳过；双后端同时故障仍需 `G4-B2` 实机证明；
- 结论：`S4-B` 的非破坏性专项通过；后续 `S4-C` 已另行完成。仍未关闭的项目属于 `S4-D` 或需要 `G4-B2` 的实机项：三个正式 task 同时替换、双协调后端故障、TLS storage 恢复和故障矩阵。当前不得据此把阶段 4 或 `G4-B2` 标记为通过。

### 9.3 S4-C：滚动更新与回滚

- [x] 通过 GoTask 执行 `release:plan -> apply -> verify`，逐节点验证 `/ready`；
- [x] 更新策略为 `parallelism: 1`、`order: stop-first`、`failure_action: pause`；
- [x] 验证单 service 更新节点本机入口存在预期短暂窗口，另外两个节点继续服务；跨 Sub2API/Caddy 的关联变更必须串行；
- [x] 制造一个可恢复的 readiness/health 失败，确认滚动暂停而不是继续推进；
- [x] 使用已记录的旧镜像及其匹配的 Config/Secret 清单执行 rollback；
- [x] 验证容器内二进制未被原地替换，三个副本最终版本一致；
- [x] 单独滚动更新模型价格 Config，不重建镜像，验证旧/新短暂并存和旧价格回滚；
- [x] 本地由当前执行者按手册完成并保存回滚证据；另一名执行者独立复现延期到生产准入。

实施记录（2026-07-27）：

- `release:verify` 原先可能在 detached `stack deploy` 返回后、旧 task 尚未全部替换时提前成功。本轮只在现有任务内增加 300 秒 rollout 等待：`paused/rollback_paused` 立即失败，并核对所有 desired-running Sub2API/Caddy task 的镜像均等于清单；未新增任务、脚本或控制面；
- 历史 `v0.1.165-ext.1` 按已记录的 source image ID、归档 SHA-256 和 node image ID 加载到三个节点，通过 GoTask 将正式 service 从 `ext.2` 实际回滚为 `ext.1`，沿用该版本原有的同一 Caddyfile、模型价格 Config 与 `app-config-v001`；随后重新滚动到 `ext.2`。最终三个容器 image ID 均为 `sha256:bb638caa30eac89bf8bb5ee6395361f941f83fc0f810150249901ba896561703`，`/app/sub2api` SHA-256 均为 `04bb1b3d8a39012a0c4e5135a950fd862b7171925b81abed70d54cbb63b5739c`；
- 完整旧清单同时改变 Sub2API healthcheck 与 Caddy upstream health，两个独立 service 会并行滚动：94 个逐秒样本中有 1 个样本同时两个入口失败，但始终至少一个入口可用。该事实形成明确规则：关联健康路径变更先用新旧版本共同支持的路径滚动应用，再单独滚动应用 healthcheck，最后滚动 Caddy upstream health，禁止一次变更两个 service；
- 按上述串行顺序重新滚动时，应用镜像阶段 97 个样本最多 1 个入口失败，应用 `/ready` healthcheck 阶段 92 个样本最多 1 个入口失败，Caddy upstream health 阶段 73 个样本全部 200；因此“另外两个节点继续服务”只对单 service 串行更新成立，不对未经编排的多 service 并行变更作保证；
- 使用只在本轮存在的错误 `app-config` Secret 让第一个新 task 无法连接 PostgreSQL，而不停止真实数据库。Swarm 进入 `paused`，`release:verify` 返回非零；9 个样本最多 1 个入口失败。重新应用正式 `app-config-v001` 后恢复成功，53 个样本最多 1 个入口失败，临时 Secret 已删除；
- 使用仅含 JSON 空白变化、业务语义不变的新模型价格 Config 完成更新并切回原 Config；更新与回滚各 93 个样本，均最多 1 个入口失败，前后镜像完全相同。最终重新引用 `sub2api-local-model-pricing-139de8a906ce`，临时 Config 已删除；
- 最终 Sub2API/Caddy 为 `3/3`，PostgreSQL/Redis 为 `1/1`；三个入口 `/ready=200`，Sub2API `UpdateStatus=completed`，healthcheck 与 Caddy upstream health 均为 `/ready`，正式 Config/Secret 引用恢复，未残留 `s4c` 临时对象。该结论只关闭 `S4-C/G4-B1`，不授权 `S4-D/G4-B2`。

### 9.4 S4-D：故障矩阵

按影响由小到大执行，每项先采集基线，失败后先保留证据：

| 场景 | 预期 | 当前状态 |
| --- | --- | --- |
| 停止单个 Sub2API task | 本节点 Caddy 短暂失败；其他节点继续服务；task 按策略恢复 | `G4-B2a` 已通过 |
| `SIGTERM`/滚动替换 | 新请求和新 WebSocket upgrade 被拒绝；已有 HTTP/SSE 按 shutdown 语义排空；已有 WebSocket 最迟在窗口到期返回 `1012`，不要求识别连接内新 turn | 待后续专项 |
| 单副本 OOM | 只影响该节点容量；记录 cgroup OOM、重启和 limiter 行为，不损坏共享状态 | `G4-B2c` 已通过；外部匿名内存压力，不代表所有业务 limiter 压测已完成 |
| 停止 `node3` | manager quorum 仍正常，容量下降一个副本；不在其他节点形成第二副本 | `G4-B2a` 已通过 |
| 停止 Redis | OAuth/共享缓存/Caddy storage 相关 readiness 符合预期；不误报健康；恢复后加载原数据 | `G4-B2b-1` 已通过暂停/恢复；进程重启/持久化恢复未验证 |
| 停止 PostgreSQL | 三个 Sub2API `/ready=503`；PostgreSQL 不漂移到空目录；恢复后重新挂载原目录 | `G4-B2b-2a` 已在 `ext.3` 通过同一容器暂停/恢复；整节点场景见下一行 |
| 停止 node2/Redis 数据节点 | 故障节点入口不可达，另外两个应用 `/health=200`、`/ready=503`；quorum 保持，Redis 新 task 仅能 Pending 且不漂移，DNS 不自动摘除 | `G4-B2b-2b-1` 已通过 |
| 停止 node1/PostgreSQL 数据节点 | node2/node3 重新形成唯一 Leader；故障节点入口不可达，另外两个应用 fail-closed；PostgreSQL 新 task 仅能 Pending 且不漂移 | 首次未通过；最小配置修正后的同场景复测已通过 |
| Caddy 重启 | 从共享 Redis storage 加载相同证书体系，不重复签发 | `G4-B2a` 已通过；Redis 不可用时未重启 Caddy，续期未验证 |
| 管理端原地更新请求 | 由 Caddy 明确拒绝；`/version` 仍正常 | `S4-A/S4-B` 已通过 |

`G4-B2a` 实施记录（2026-07-27）：

- 强制结束 node3 上单个 Sub2API task `z0mlxhbc608deyxamncztk60a` 后，该 task 于 `2026-07-27 04:08:50.512755854 UTC` 以 exit code 137 失败；Swarm 创建的 `b25klve6g0p5einofwr043jvs` 于 `04:09:01.102583015 UTC` 进入 Running，约耗时 10.6 秒，新容器恢复 healthy。故障采样期间 node1/node2 始终返回 200，仅 node3 短暂返回 502/503，最终三个入口 `/ready=200`；
- 停止 node3 后，Swarm 在 15 秒内将其标记为 `Down/Unreachable`，node1 继续为 Leader、node2 保持 Reachable。Sub2API/Caddy 从 `3/3` 降为 `2/3`，node1/node2 各保持一个 global task，没有在剩余节点补第二副本；PostgreSQL/Redis 保持 `1/1`。重新启动 node3 后约 24 秒恢复为 Ready/Reachable，两个 global task 与本机 `/ready` 均恢复；
- 强制结束 node3 上单个 Caddy task `7ubcmz17mnpfkmi3xynwx26rq` 后，该 task 于 `2026-07-27 04:13:50.923163934 UTC` 以 exit code 137 失败；Swarm 创建的 `cut05dm2d2aidxaeealiv0evs` 于 `04:13:56.109691979 UTC` 进入 Running，约耗时 5.2 秒。故障采样期间 node1/node2 始终返回 200，仅 node3 短暂不可达，最终三个入口均恢复 200；
- Caddy 重启前后三个入口的叶证书 serial 均为 `6A756405F963CC3B7D3310DCAF348F5B`，SHA-256 指纹均为 `40:FD:12:CF:C3:3C:C4:B8:45:80:75:AD:1F:09:91:C1:4E:A2:5D:FA:50:C5:F1:C2:3E:5C:C1:D5:A6:F3:15:7A`；Redis DB 1 的 Caddy storage key 数保持 15，key name set SHA-256 保持 `8c59bdb6c96e954ab0b28d80c55c18c09e7ad3ed57992d7231c7a67c91a72ca8`，新 Caddy 日志未出现证书签发事件；
- 最终 `release:verify ENV=local` 通过：三个 manager 均 Ready，node1 为唯一 Leader，node2/node3 Reachable；Sub2API/Caddy 为 `3/3`，PostgreSQL/Redis 为 `1/1`。本记录只证明单 task、单 manager 节点和 Caddy 从正常共享 storage 重启的恢复行为，不证明 Redis 故障恢复、证书续期协调、OOM 或 migration 失败行为。

`G4-B2b-1` Redis 中断恢复记录（2026-07-27）：

- 基线 `release:verify ENV=local` 通过；Redis task 为 `qfhw450m6d8e30ambjwsi6k4n`，node2 容器为 `17de281ecc86` 且 health=healthy。Caddy Redis DB 1 有 15 个 storage key，key name set SHA-256 为 `8c59bdb6c96e954ab0b28d80c55c18c09e7ad3ed57992d7231c7a67c91a72ca8`；三个入口的证书 serial/指纹与 G4-B2a 基线一致；
- `2026-07-27 04:37:26.445889019 UTC` 暂停该 Redis 容器，`04:37:51.499959160 UTC` 在退出 trap 保护下解除暂停，中断约 25.05 秒。中断期间三个 Sub2API 均保持直连 `/health=200`，直连 `/ready=503`；三个 HTTPS 入口均成功完成 TLS 握手并返回 503，说明 Caddy 继续使用已加载证书，同时 active health 不会把 Redis 依赖异常的本机应用误报为 ready；
- Redis 在解除暂停后约 0.09 秒恢复 `PONG`；约 1.0 秒后的完整采样中三个直连 `/ready` 与 HTTPS 入口均恢复 200，Docker health 于 `04:38:01.571153206 UTC` 恢复 healthy，约耗时 10.1 秒。Redis task/container ID 均未变化，没有触发 Swarm task 替换；
- 恢复后 `release:verify ENV=local` 再次通过；Sub2API/Caddy 为 `3/3`，PostgreSQL/Redis 为 `1/1`。三个 Caddy task 未重启，证书 serial/指纹不变；Caddy Redis DB 1 仍为相同 15 个 key 和相同 key name set SHA-256，日志中的证书签发事件、Redis storage error 均为 0；Sub2API panic/fatal 为 0；
- 应用 Redis DB 0 的动态 key 数从 121 变为 122；该库包含带 TTL 的缓存、锁和运行态 key，因此不把瞬时 key 数当作完整性校验。本轮没有创建业务测试实体。该结果只证明同一 Redis 进程的短时不可用、readiness 失败与恢复，不证明进程重启/AOF 重放、数据卷恢复、Redis 不可用时重启 Caddy、真实 OAuth 事务或证书续期行为。

`G4-B2b-2a` PostgreSQL 中断恢复记录（2026-07-27，门槛未通过）：

- 基线 `release:verify ENV=local` 通过；PostgreSQL task 为 `b5ysani4aye7gl2gbpxwv03v6`，node1 容器为 `81c5e2921ae8` 且 health=healthy，volume 为 `sub2api-local_postgres_data:/var/lib/postgresql/data`；`schema_migrations` 为 236 条、236 个唯一 filename、0 个空 checksum；
- `2026-07-27 04:46:28.329936310 UTC` 暂停该 PostgreSQL 容器，`04:46:53.352972624 UTC` 在退出 trap 保护下解除暂停，中断约 25.02 秒。中断期间三个 Sub2API 始终保持直连 `/health=200`，但直连 `/ready` 在连续多轮 4 秒客户端期限内均超时并返回采样状态 `000`，没有按门槛返回 503；Caddy active health 在过渡后让三个 HTTPS 入口稳定返回 503，未误报 ready；
- 运行时代码已设置 `dependencyProbeTimeout=2s`，但当前使用的 `lib/pq v1.10.9` 在取消路径中另有 10 秒 cancel dial timeout；本次现象与复用连接上的 `PingContext` 无法在 PostgreSQL 进程暂停时及时返回一致。后续已用阻塞 pinger 单元测试固定该行为，并在独立授权下实施最小修补；现场记录本身仍只证明原 `ext.2` 运行态存在偏差；
- 解除暂停后 PostgreSQL 约 0.25 秒恢复 `pg_isready`；三个直连 `/ready` 恢复 200，三个 Caddy 入口约 3.3 秒内全部恢复 200，Docker health 于 `04:47:03.390196748 UTC` 恢复 healthy，约耗时 10.0 秒；
- 恢复后 `release:verify ENV=local` 再次通过；PostgreSQL task/container/volume、三个 Sub2API task 和三个 Caddy task 均未变化，`schema_migrations` 仍为 `236/236/0`，Sub2API panic/fatal 为 0，最终 Sub2API/Caddy 为 `3/3`、PostgreSQL/Redis 为 `1/1`；
- 本轮没有修改代码、配置、service spec、Secret、Config、volume 或业务数据，也没有触发 migration。该结果证明 Caddy/Swarm 外层 probe 会 fail-closed 且环境可恢复，但由于直连 `/ready` 未受 2 秒预算约束，`G4-B2b-2a` 当前不得标记通过。

`G4-B2b-2a-fix` 仓库最小修补记录（2026-07-27，代码提交 `593a261d7`，现场复测待授权）：

- 修改严格限制为 `backend/extends/lifecycle/manager.go` 与 `manager_test.go`；保留公开构造函数 `NewManager(*sql.DB, *redis.Client)`，仅在包内增加窄 `databasePinger` 接口和测试 fake，没有新增实体、配置项、依赖、功能开关或通用框架；
- PostgreSQL readiness 同一时刻最多保留一个共享 in-flight probe；每个 `/ready` caller 仍按自己的 context deadline 返回。共享 probe 使用独立 2 秒 context，避免最先断开的 caller 取消其他 caller 正在等待的探测；即使 driver 的取消路径继续阻塞，也只占用一个受控 goroutine；
- 阻塞 pinger 测试覆盖三个并发 caller、首个 caller 取消后的跟随请求、底层 `PingContext` 仅调用一次，以及释放探测后的恢复。`go test ./extends/lifecycle -count=1`、`go test -race ./extends/lifecycle -count=1`、目标并发测试 `-count=100`、`go vet ./extends/lifecycle`、相关包测试和 `go test ./... -count=1` 均通过；
- 该修补提交形成时 `backend/extends/VERSION` 仍为 `ext.2`，未创建新 tag 或镜像，未上传、部署或修改 Multipass 运行态；后续候选构建由独立授权处理。因此本记录只关闭仓库代码与测试子门槛，`G4-B2b-2a` 仍须在候选版本部署后重复原 PostgreSQL 暂停/恢复场景，才能重新判定是否通过。

`G4-B2b-2a-candidate` 本地 ARM64 候选记录（2026-07-27，候选形成时节点部署与现场复测待授权）：

- `593a261d7` 提交审核无阻断；公开构造函数未变，包内窄接口、互斥状态与 caller/probe context 分离符合最小修补边界。随后仅将 `backend/extends/VERSION` 从 `ext.2` 提升为 `ext.3`，形成版本提交 `6c859d2d83e03c49fb49a53e530932d7a6c789d7`；上游版本仍为 `0.1.165`，未创建或推送 `v0.1.165-ext.3` tag；
- `go mod verify` 与 `go test ./... -count=1` 通过；使用固定 `Version=0.1.165-ext.3`、上述 commit 和 commit date 构建 `linux/arm64` 本地镜像 `sub2api-local/sub2api:v0.1.165-ext.3-arm64`。source image ID 为 `sha256:03e01bbd24c1818ac1f8ad9ec6413969ed9e6e69a524cb2795f993ed756da6aa`，`/app/sub2api` SHA-256 为 `c6d73fc00d060cf1d04ae0ffc3f76796b1c679bd14205692ad3f73c63e4e8b65`；
- 镜像平台、OCI version/revision/source 标签及 `--version` 输出均与构建输入一致；按现有 `docker image save` 路径两次流式计算的归档 SHA-256 均为 `43d69c5fa76eb0f3c809a97251f2eee3477c04cb5324d6449fea6b6bb67b1f6c`；
- 候选形成时只保留在 macOS 本机 Docker。三个 Multipass 节点均不存在该 tag，活动 service 当时仍为 `sub2api-local/sub2api:v0.1.165-ext.2-arm64`；未修改 `deploy/cluster/env/local-arm64/cluster.env`，未上传 GHCR、未分发、未部署、未运行 migration，也未重复故障注入。仓库 VERSION 与活动清单在候选审核期有意不一致，当时 checkout 的 `validate:stack ENV=local` 会按门禁拒绝，部署授权前不得通过改校验逻辑绕过。

`G4-B2b-2a-deploy-retest` 三节点部署与 PostgreSQL 复测记录（2026-07-27，已通过）：

- 本机 `sub2api-local` Docker context 元数据缺失后，只把宿主机既有 SSH 公钥加入 node1 并重建 `ssh://ubuntu@192.168.252.2` 管理 context；未启用密码 SSH、未暴露 Docker TCP daemon，也未改变任何 service。发布操作仍由 node1 的固定 GoTask 工作副本执行；
- 已核验候选归档以 source image ID `sha256:03e01bbd24c1818ac1f8ad9ec6413969ed9e6e69a524cb2795f993ed756da6aa`、归档 SHA-256 `43d69c5fa76eb0f3c809a97251f2eee3477c04cb5324d6449fea6b6bb67b1f6c` 分发到三个节点，三节点 node image ID 均为 `sha256:fd867fc19da56a25bae98930d2186159f3650a83cc5cefb99164ae4951f01a6f`，容器内 `/app/sub2api` SHA-256 均为 `c6d73fc00d060cf1d04ae0ffc3f76796b1c679bd14205692ad3f73c63e4e8b65`；活动清单提交 `3608d6c7b` 固定上述身份；
- `release:plan -> release:apply -> release:verify` 完成 `ext.2 -> ext.3` 受控滚动，Swarm update 从 `2026-07-27 05:22:26.365631225 UTC` 到 `05:23:58.686147025 UTC` 完成。node1/node2/node3 新 task 分别为 `kxnoaqpr87qg`、`21sv0gl68zcv`、`w9r8eytyd5k7`，均为 healthy、报告 `0.1.165-ext.3` 与 commit `6c859d2d83e03c49fb49a53e530932d7a6c789d7`；三节点直连 `/health`、`/ready` 与 HTTPS 均为 200，migration 保持 `236/236/0`；
- 入口采样覆盖 node1 与 node2 的替换窗口，期间最多同时一个 HTTPS 入口失败；采样在 node3 替换前结束，因此不把该采样外推为覆盖完整滚动。node3 由最终 healthy task、逐节点 200 与 `release:verify` 单独确认；
- PostgreSQL 容器 `81c5e2921ae8` 在退出 trap 保护下从 `05:24:39.773162483 UTC` 暂停至 `05:25:04.865466972 UTC`，约 25.09 秒。期间三个节点 `/health` 均为 200；每节点连续三次、共九次直连 `/ready` 均返回 503，耗时范围约 2.0015–2.0653 秒，三个 HTTPS 入口均返回 503，原先连续 4 秒得到 `000` 的问题没有复现；
- 解除暂停后约 15.75 秒取得首个完整恢复样本：PostgreSQL 接受连接且 healthy，三个直连 `/health`、`/ready` 与 HTTPS 均恢复 200。PostgreSQL task `b5ysani4aye7gl2gbpxwv03v6`、容器、volume、三个 Sub2API/Caddy task 均未替换，migration 仍为 `236/236/0`，近 10 分钟 Sub2API panic/fatal 为 0，最终 `release:verify` 通过；
- 本次只关闭“同一 PostgreSQL 容器短时暂停/恢复”的 readiness 门槛，不覆盖 PostgreSQL 进程重启、volume 重挂载、node1 停止、备份恢复、OOM、migration 失败或生产故障域。未创建 `v0.1.165-ext.3` Git tag，未上传 GHCR，因复测通过未执行 `ext.2` 回滚。

`G4-B2b-2a-tag` ext.3 标签闭环记录（2026-07-27，已完成）：

- annotated tag `v0.1.165-ext.3` 的 tag object 为 `de000a7f6ed506b76b10384da8301dc18c485637`，peel 后固定到版本提交 `6c859d2d83e03c49fb49a53e530932d7a6c789d7`；本地与 `origin` 的 peeled commit 已核对一致；
- 标签创建前确认目标提交位于当前 `main` 历史中，两个 VERSION 组合为 `0.1.165-ext.3`，本地/远端不存在同名标签，`backend/go.mod` 不含 `replace`；
- 只推送 Git tag。release workflow 仅接受 `workflow_dispatch`，本次 tag push 未触发 GHCR 构建；未上传镜像、未创建 GitHub Release、未修改集群运行态或分支内容。

`G4-B2b-2b-review` 数据节点故障执行前只读审查（2026-07-27，已完成；审查时实际故障未授权，node2 后续已通过）：

审查结论为 **Approved**。不需要修改 Go 代码、Stack、GoTask、label、service spec、Secret/Config 或 volume，也不新增脚本、任务、控制面和实体；实际故障必须拆成 `G4-B2b-2b-1` node2/Redis 与 `G4-B2b-2b-2` node1/PostgreSQL 两次独立授权，先低控制面风险的 node2，完整恢复后才可考虑 node1。

只读基线：

- 三个节点均为 `Ready/Active` manager，node1 当前为 Leader，node2/node3 为 Reachable；quorum 为 2。Sub2API/Caddy 为 `3/3`，PostgreSQL/Redis 为 `1/1`，`release:verify ENV=local` 通过；node1 停止时既有 `sub2api-local` Docker context 会不可用，已确认可改用 `multipass exec node2 -- docker ...` 只读观察剩余控制面；
- PostgreSQL 当前 task/container 为 `t4ns8vvywx85`/`4a50cd8f4a12`，仅 node1 带 `postgres=true`。local volume `sub2api-local_postgres_data` 创建于 `2026-07-27T00:28:23+08:00`，mountpoint 为 `/var/lib/docker/volumes/sub2api-local_postgres_data/_data`，基线 device/inode 为 `2049/302196`；PostgreSQL `system_identifier=7666874411637911585`，migration 为 236 条、236 个唯一 filename、0 个 null checksum、0 个空 checksum；
- Redis 当前 task/container 为 `qv9imdixga3m`/`9cf548417e10`，仅 node2 带 `redis=true`。local volume `sub2api-local_redis_data` 创建于 `2026-07-27T00:14:54+08:00`，mountpoint 为 `/var/lib/docker/volumes/sub2api-local_redis_data/_data`，基线 device/inode 为 `2049/299241`；`PONG`、RDB/AOF 状态正常，DB 1 为 15 个 Caddy storage key，key name set SHA-256 为 `8c59bdb6c96e954ab0b28d80c55c18c09e7ad3ed57992d7231c7a67c91a72ca8`。DB 0 含 TTL/锁/缓存，瞬时 key 数不作为不变量；
- 三个直连 `/health`、`/ready` 均为 200；三个 HTTPS 入口的叶证书 serial 均为 `6A756405F963CC3B7D3310DCAF348F5B`，SHA-256 指纹均为 `40:FD:12:CF:C3:3C:C4:B8:45:80:75:AD:1F:09:91:C1:4E:A2:5D:FA:50:C5:F1:C2:3E:5C:C1:D5:A6:F3:15:7A`。

未来实际执行的共同保护与顺序：

1. 每个子场景开始前重新采集上述动态 task/container、leader、volume 和数据不变量；发现 rollout、quorum、health、placement、磁盘或工作树异常即不开始。
2. 在 macOS 宿主机的同一操作 shell 中先注册 `EXIT/INT/TERM/HUP` 恢复 trap，再以 `nohup` 启动 60 秒后调用固定路径 `/usr/local/bin/multipass start <node>` 的一次性恢复 watchdog，并以 `kill -0` 确认存活；然后执行不带 `--force` 的普通 `multipass stop <node>`。node1 必须使用手册中硬编码的专用命令块，不能编辑 node2 模板。取得预期故障证据后立即人工启动，不为等待固定时长延长中断；只有 `multipass start` 成功、`multipass info` 为 `Running` 且 `multipass exec node1 -- true` 成功后，才能终止 watchdog、撤销 trap 并清理 `/tmp` 日志。任一步失败时保留 watchdog，并让 `EXIT` trap 再次恢复。
3. 禁止 `multipass stop --force`、同时停止第二个 manager、`docker node ... drain`、修改 label/service spec/Secret/Config、删除或重建 volume、prune、stack remove、手工迁移数据服务、重复 bootstrap 或业务写入测试。
4. 先单独执行 node2/Redis：故障后 30 秒内 node2 应进入 `Down/Unreachable`，node1/node3 保持 quorum 与唯一 Leader；node2 入口预期不可达，node1/node3 直连 `/health=200`、`/ready` 应在约 3 秒内返回 503，HTTPS 可使用已加载证书返回 503。Redis 的新 desired task 必须因唯一 placement 无可用节点而保持 Pending 且没有 NODE，两个存活节点各只保留一个 Sub2API/Caddy task。不可达节点的旧 task 可能保留最后已知 `Running`，global service 的 desired 数也会随可用节点变化，因此不得把 `docker service ls` 的精确 `REPLICAS` 比值作为 liveness 门槛。期间禁止重启剩余 Caddy；恢复后 node2 Caddy 从已恢复的 Redis storage 冷启动属于本场景，但不外推为“Redis 持续不可用时 Caddy 冷启动”。
5. node2 完整恢复后才可单独执行 node1/PostgreSQL；该场景只有在最小配置修正已应用并完成常态验证后才能另行授权复测。node1 stop 返回后 30 秒内，node2/node3 必须维持 quorum 且只有一个 Leader；50 秒内，PostgreSQL 的新 desired task 必须因唯一 placement 无可用节点而 Pending、没有 NODE 且不得漂移。任一门槛未满足即恢复并判失败；Pending 一出现即人工恢复，不等待 50 秒或 60 秒 watchdog。不要求 node1 恢复后重新成为 Leader；Redis 保持在 node2。node1 入口预期不可达；node2/node3 直连 `/health=200`、`/ready` 应在约 3 秒内返回 503。node2 没有 GoTask 部署副本或本地 CA，故障期只能用其原生 Docker CLI 和 `/usr/bin/timeout` 观察控制面；HTTPS 使用 `curl -k` 时必须同时核对预先记录的 serial/指纹，恢复后再由 node1 的 `release:verify` 完成 CA 验证。与 node2 场景相同，验收读取 task-level desired/current state、NODE 和调度错误，不使用汇总 `REPLICAS` 比值推断不可达节点上的进程仍存活。新的拆分门槛不追认历史执行通过。
6. 每个场景恢复后允许数据 service、Sub2API 和 Caddy 产生新 task/container ID，但必须重新使用同名 local volume 和原节点；等待三个 manager Ready/Reachable、唯一 Leader、Sub2API/Caddy `3/3`、PostgreSQL/Redis `1/1` 且 healthy、三个直连及 HTTPS `/ready=200`，最后 `release:verify ENV=local` 通过。
7. PostgreSQL 恢复后 `system_identifier` 必须不变，migration 必须仍为 `236/236/0/0`；Redis 必须 `PONG`、RDB/AOF 正常，DB 1 key 数和 key-name 摘要不变；三个证书 serial/指纹不变且无新签发。Sub2API/PostgreSQL/Redis/Caddy 日志不得出现 panic、fatal、corruption 或恢复后持续错误。

立即恢复并判失败的门槛：剩余 manager 少于两个或 30 秒内未形成唯一 Leader、node1 场景在 50 秒内未出现符合 placement 的 PostgreSQL 无 NODE/Pending task、另一节点异常、数据 service 在非原节点启动或出现新空 volume、剩余应用 `/health` 非 200、`/ready` 挂起或误报 200、watchdog 未存活、节点启动后 120 秒内数据 service 未 healthy、300 秒内未达到完整基线、system identifier/migration/volume/Caddy storage/证书不变量异常，或恢复需要修改 spec/Secret/Config/label。出现数据不变量异常时只保留证据，不执行 stack deploy、service force-update、bootstrap、migration 修复、volume 替换、重新加入 Swarm 或第二个子场景。

该方案只验证普通 `multipass stop` 的受控关机与原虚拟磁盘恢复，不使用可能损坏数据的 `--force`，因此不证明断电/宿主崩溃、磁盘损坏、跨节点恢复、备份恢复、自动故障转移、DNS 摘除、生产 HA 或 RPO/RTO。

`G4-B2b-2b-1` node2/Redis 数据节点停止/恢复记录（2026-07-27，已通过）：

- 执行前 `release:verify ENV=local`、三个入口、quorum、placement 与数据身份均通过。Redis task/container 为 `qfhw450m6d8e30ambjwsi6k4n`/`17de281ecc86`，local volume 为 `sub2api-local_redis_data`，Mountpoint `/var/lib/docker/volumes/sub2api-local_redis_data/_data`，device/inode `2049/299241`；Redis `PONG`、RDB/AOF 正常，Caddy DB 1 为 15 个 key 且 key-name-set SHA-256 为 `8c59bdb6c96e954ab0b28d80c55c18c09e7ad3ed57992d7231c7a67c91a72ca8`。PostgreSQL `system_identifier=7666874411637911585`，migration 为 `236/236/0/0`；
- 首次受控窗口从 `06:27:05Z` 开始，node2 停止后两个存活节点 `/health=200`、直连 `/ready=503`（约 2.03–3.00 秒）、HTTPS `/ready=503`，node2 直连和 HTTPS 均不可达；node1 保持唯一 Leader、node3 Reachable。因在控制面收敛前即按保护原则恢复，该窗口只保留为入口证据；完整恢复和数据复核通过后，在同一授权范围内复测；
- 复测从 `06:29:36Z` 开始，node2 在 15 秒内进入 `Down/Unreachable`，node1/node3 保持 quorum。两个存活节点 `/health=200`，直连 `/ready=503` 约 3.00 秒，HTTPS 返回 503，node2 两个入口不可达。Swarm 为 Redis 创建 `qv9imdixga3m4mryny8arnu2a`，该 task 没有 NODE 并因唯一 placement 无可用节点而 Pending，没有漂移或创建空 volume；
- 故障期 `docker service ls` 实际显示 Sub2API/Caddy `3/2`、Redis `1/1`：不可达节点旧 task 保留最后已知 `Running`，global service desired 数变为两个可用节点。该汇总值不能证明故障节点进程存活，后续门槛已改为 task-level desired/current state、NODE、调度错误及真实入口状态；
- 在复测开始 35 秒时人工启动 node2，49 秒时 `multipass start` 返回，60 秒 watchdog 未触发且已清理。恢复后 Redis task/container 为 `qv9imdixga3m4mryny8arnu2a`/`9cf548417e10`，从原 AOF 在约 0.034 秒内加载完成；原 volume 名称、创建时间、Mountpoint、device/inode 均不变，`PONG`、RDB/AOF、15 个 Caddy key 及摘要均通过；
- node2 的 Sub2API/Caddy task 按预期替换，node1/node3 task 未替换。三个 manager 恢复 Ready/Reachable，四项 service 恢复 `3/3、3/3、1/1、1/1`；三个直连 `/health`、`/ready` 与 HTTPS `/ready` 均为 200，证书 serial/指纹不变，PostgreSQL identity/migration 不变。相关日志中 panic/fatal/corruption、Caddy 新签发或 Redis storage error 均为 0，最终 `release:verify ENV=local` 通过；
- 结论仅覆盖普通 `multipass stop/start node2`、原虚拟磁盘与原 local volume 恢复；不覆盖 `--force`、断电、磁盘损坏、VM 重建、跨节点/备份恢复、自动故障转移、Redis 持续不可用时 Caddy 冷启动、DNS 摘除或生产 HA。

`G4-B2b-2b-2` node1/PostgreSQL 执行前只读复审（2026-07-27，已通过；以下为执行前历史状态）：

- 复审期间没有停止节点或 service，也没有修改 Go、Stack、GoTask、label、service spec、Secret/Config、volume、镜像或运行态。仓库 `main` 与 `origin/main` 一致，活动提交为 `d7f424c141a7f99b6d2d06614d80e8b886b52f96`；正确入口上的 `release:verify ENV=local` 通过；
- node1/node2/node3 均为 `Ready/Active` manager，node1 当前为唯一 Leader。宿主机 `sub2api-local` context 为 `ssh://ubuntu@192.168.252.2`，node1 停止后必然不可用；node2 已只读验证可通过原生 `/usr/bin/docker` 管理 Swarm，且 `/usr/bin/timeout`、`curl`、`openssl` 可用，但 node2 没有 `task`、`/home/ubuntu/sub2api-fork/deploy/cluster` 或本地 CA 文件；
- PostgreSQL service spec version 为 `707`，placement 固定 `node.labels.postgres == true` 与 `aarch64`，当前 task/container 为 `t4ns8vvywx85wdwon1dksg524`/`4a50cd8f4a12`。named volume `sub2api-local_postgres_data` 的创建时间为 `2026-07-27T00:28:23+08:00`，Mountpoint 为 `/var/lib/docker/volumes/sub2api-local_postgres_data/_data`，device/inode 为 `2049/302196`；`pg_isready` 通过，`system_identifier=7666874411637911585`，migration 为 `236/236/0/0`；
- Redis 位于 node2 且 `PONG`；node2/node3 的直连 `/health`、`/ready` 与临时 `curl -k` HTTPS 均为 200，叶证书 serial/指纹与既有基线一致。故障期不在 node2 运行 `release:verify`，而以有界 Docker 命令、入口状态和 `openssl` 指纹取证，node1 恢复后再执行完整 CA 验证；
- 独立规格复审最初发现两个阻断项：node1 依赖编辑 node2 模板的五处替换，以及 `multipass start` 后未确认来宾可用就撤销 watchdog/trap。现已改为硬编码 node1 的专用命令块，并要求 `start` 成功、状态为 `Running`、`multipass exec node1 -- true` 成功后才撤销两层保护；任一步失败则保留 watchdog 并触发 `EXIT` 恢复；
- node1 故障期的 30 秒门槛同时覆盖 node2/node3 形成唯一 Leader，以及 PostgreSQL 新 desired task 无 NODE、因唯一 placement 无可用节点而 Pending；不读取误导性的 `service ls` 汇总比值。数据 service healthy 为节点启动后 120 秒门槛，完整基线为 300 秒；
- 修正后复审结论为 **Approved**。该结论只说明实际执行方案可以进入独立授权判断，不授权停止 node1，也不授权 `--force`、drain、修改 spec/label/volume、OOM、migration 故障、DNS 或生产变更。

`G4-B2b-2b-2` node1/PostgreSQL 数据节点停止/恢复记录（2026-07-27，已执行，未通过）：

- 执行前 `release:verify ENV=local`、三个 manager、四项 service、三个直连/HTTPS 入口、placement 与数据身份均通过。PostgreSQL task/container 为 `t4ns8vvywx85wdwon1dksg524`/`4a50cd8f4a12`，named volume `sub2api-local_postgres_data` 的创建时间、Mountpoint、device/inode `2049/302196` 均与复审基线一致；`system_identifier=7666874411637911585`，migration 为 `236/236/0/0`；
- `06:52:03Z` 建立硬编码 node1 的 60 秒 watchdog 与退出 trap，普通 `multipass stop node1` 于 `06:52:07Z` 返回。node2 在约 12 秒后成为唯一 Leader，约 17 秒时 node2/node3 均为 Ready；但 30 秒内 PostgreSQL 仍显示 node1 上旧 task 的最后已知 `Running`，没有出现要求的无 NODE/Pending task，因此按既定门槛判失败并立即恢复，没有延长窗口或重复执行；
- 故障期 node1 入口不可达；node2/node3 `/health=200`，直连 `/ready=503` 分别约 2.08/2.04 秒，HTTPS `/ready=503`，叶证书 serial/指纹与基线一致。`06:52:43Z` 人工启动 node1，确认 start 成功、VM 为 Running、来宾可执行后撤销保护；watchdog 日志为空，未触发自动启动；
- PostgreSQL 恢复后的 task/container 为 `zf8yvth1nrkna4iehx79g6qkj`/`8139962ea6fc` 且 healthy，仍挂载原 named volume。volume 创建时间、Mountpoint、device/inode、`system_identifier` 与 migration 全部不变；Redis task/container、AOF/RDB、15 个 Caddy storage key 及摘要也不变，三个 manager、Caddy `3/3`、PostgreSQL/Redis `1/1` 和证书均恢复；
- 恢复期间 node2/node3 的既有 Sub2API task 分别于 `06:53:30Z`/`06:53:24Z` 变为 `unhealthy` 后以 `exit 0`、task `Complete` 结束。服务仍使用 `restart_policy.condition=on-failure`，没有自动创建替代 task；300 秒完整恢复门槛后 Sub2API 仍为 node1 `1/1`，node2/node3 直连 `:8080` 不可达、HTTPS `/ready=503`，最终 `release:verify ENV=local` 因运行 task 数不足而失败；
- 当前没有数据损坏或 service spec/label/volume 漂移证据，相关应用/数据/Caddy 日志的 panic、fatal、corruption、storage error 和证书新签发模式均为 0。但故障授权范围内没有完成环境恢复，故本场景结论必须为 **未通过**。按停止门槛未执行 Stack 重部署、service force-update、restart policy 修改、第二节点重启、复测、OOM 或 migration 故障。

`G4-B2b-2b-2-recovery` 最小环境恢复记录（2026-07-27，已通过）：

- 执行前重新核对三个 manager、固定 Sub2API image ID、Config/Secret、placement、resource、restart、update/rollback、global mode 与 host-mode endpoint。只对 `sub2api-local_sub2api` 执行一次受控 `docker service update --force --detach=false`；ForceUpdate generation 从 0 增至 1，其他已登记字段保持不变，没有 Stack 重部署、源码/配置修改或其他 service/节点操作；
- rollout 从 `07:04:22Z` 到 `07:05:49Z` 按 `parallelism=1`、`delay=10s`、`stop-first`、`monitor=30s` 完成。期间 CLI 曾短暂报告 host-mode port in use 和 task failure 检测，但既有 `failure_action=pause` 未触发，最终 `UpdateStatus=completed`；新 task 为 node1 `sfb1tefnklak28qjnniczxgb3`、node2 `i8f63oj42ylw3a2z3gwjzax70`、node3 `q0jmzdsi8ickapy8ixthzz695`，均为 healthy；
- 最终三个 manager Ready/Reachable、node2 为唯一 Leader，Sub2API/Caddy `3/3`、PostgreSQL/Redis `1/1`。三个直连 `/health`、`/ready` 与 HTTPS `/ready` 均为 200，`release:verify ENV=local` 通过；PostgreSQL task/container 与原 volume、`system_identifier=7666874411637911585`、migration `236/236/0/0` 不变，Redis `PONG`、AOF/RDB、15 个 Caddy key 及摘要、证书 serial/指纹也不变，告警模式计数为 0；
- 该授权只关闭当前环境恢复，不把 `G4-B2b-2b-2` 改为通过，也不授权 restart policy/healthcheck/watchdog 修改、故障复测、其他 force-update、OOM 或 migration 故障。

`G4-B2b-2b-2-fix-review` 配置层最小修正审查（2026-07-27，已完成并通过）：

- 现场时序重建确认：node1 stop 于 `06:52:07Z` 返回，node2 约 12 秒后成为 Leader、约 17 秒时两个存活 manager Ready；node1 heartbeat 到 `06:52:47Z` 才过期，PostgreSQL 新 task 随后创建并于 `06:53:05Z` Running。原规则把 Leader 与 PostgreSQL Pending 都压在 30 秒内，与本地实测约 40 秒才形成新 desired task 的前提冲突，故只能保留为历史失败规则，不能用于复测；
- node3/node2 的旧 Sub2API container 分别于 `06:53:24Z`/`06:53:30Z` 以 `exit 0` 结束，task 为 `Complete`。Docker 的 `on-failure` 仅覆盖非零退出，现有 condition 因而不会补齐 global service 的这两个 desired task；恢复使用的单次 force-update 只能恢复当时环境，不能消除该配置缺口；
- 最小修改白名单只有两项：在 `deploy/cluster/stacks/sub2api.yml` 仅把 Sub2API 的 `restart_policy.condition` 从 `on-failure` 改为 `any`；在既有 `deploy/cluster/taskfiles/validate.yml` 的 `validate:stack` 内增加渲染断言，确认 Sub2API 为 `any` 且不误改其他 service。保留 `delay=5s`、`max_attempts=3`、`window=60s`、`/ready` healthcheck、`interval=10s`、`timeout=3s`、`retries=6`、`start_period=30s`、其他 service 及全部 Go 代码不变；不新增脚本、任务、实体、开关或恢复控制面，不进入 `backend/extends`；
- 复测门槛拆为：停止返回后 30 秒内两个存活 manager 保持 quorum 且只有一个 Leader；50 秒内 PostgreSQL 新 desired task 无 NODE、因唯一 placement 无可用节点而 Pending，且不得漂移；任一门槛失败立即恢复，Pending 一出现即人工恢复，不等待定时器。保留 60 秒宿主机 watchdog、节点恢复后 120 秒数据 service healthy 与 300 秒全栈基线；这些新门槛仅用于配置修正并应用后另行授权的复测，不追认历史执行通过；
- 独立规格复审结论相同：不调整 healthcheck 或 retries，不改其他 restart 参数；复测额外观察持续依赖故障下的 task churn，并重新验证普通滚动、回滚与 `release:verify`。审查结论为 **Approved**，但本轮没有修改 Stack/源码/运行态，也没有执行故障。

`G4-B2b-2b-2-fix/fix-deploy/retest` 实施记录（2026-07-27，已完成并通过）：

- 静态修正严格限于两个部署文件：Stack 中仅把 Sub2API `restart_policy.condition` 改为 `any`；既有 `validate:stack` 增加四个 service 的渲染断言，要求 Sub2API 为 `any`，Caddy/PostgreSQL/Redis 仍为 `on-failure`。先修改断言得到预期 RED：`sub2api=on-failure expected=any`；再修改 Stack 后 local 与 production-amd64 渲染均通过，`git diff --check` 通过。未修改 Go、healthcheck、其他 restart 参数，也未新增任务、脚本、实体、开关或 `extends` 代码；独立规格审查结论为 **Approved**；
- 运行态只执行一次 `docker service update --restart-condition any`，没有重部署 Stack、force-update 或修改其他 service。Sub2API rollout 于 `07:35:01Z–07:36:42Z` 串行完成，ForceUpdate 保持 1；滚动期间只在被替换节点出现短暂 502/503，另外两个入口持续 200，最终 task 为 node1 `8m3n69gaco94`、node2 `fpdibey2bg2`、node3 `s3dvdwf1plp2`，三个直连与 HTTPS 均为 200，`release:verify` 通过；
- 首次复测尝试的入口采样使用无参数 `wait`，误等待了同一 shell 的 60 秒 watchdog；watchdog 按设计自动恢复 node1，观测跨过 50 秒门槛，因此该次明确记为**无效执行**而非通过。恢复后约 61 秒三个应用副本和入口重新为 200，数据身份无漂移；
- 修正采样器后重新执行同一场景：node1 stop 于 `07:42:19Z` 返回，node2/node3 的 quorum 与唯一 Leader 在 0 秒确认；15 秒时 PostgreSQL 新 task `2e6vjbedzxd4` 无 NODE、因唯一 placement 无可用节点而 Pending，立即人工启动 node1。故障期两个存活节点 `/health=200`，直连 `/ready=503` 约 2.01–2.04 秒，HTTPS `/ready=503`；`07:42:53Z` 来宾恢复并安全撤销 watchdog；
- 恢复观测起点 PostgreSQL 已 healthy，12 秒内三个 manager、Sub2API/Caddy `3/3`、PostgreSQL/Redis `1/1` 与三个入口全部恢复。Sub2API 的 `condition=any` 自动补齐 node2/node3 正常退出的 task；四项 service spec 摘要与故障前一致，PostgreSQL volume device/inode `2049/302196`、`system_identifier=7666874411637911585`、migration `236/236/0/0`，Redis device/inode `2049/299241`、RDB/AOF、15 个 Caddy key 及摘要，证书 serial/指纹均不变；最终 `release:verify` 与告警日志门槛通过。

`G4-B2c` 实施记录（2026-07-27，已完成并通过）：

- 单副本 OOM 仅作用于 node3 当前 Sub2API cgroup。第一次容器内 `awk` 在触发 cgroup 限制前自行返回 `out of memory`，`memory.events` 仍全为 0、容器和三个入口持续 200，故不计有效场景。随后把一个宿主机 Python 匿名内存分配器移动到同一临时 cgroup，并仅对该 cgroup 启用 `memory.oom.group=1`；
- 有效注入于 `07:46:33Z` 达到 `memory.peak=2147483648`，记录 `max=20、oom=1、oom_kill=3、oom_group_kill=1`。原 task/container `tn8hrlyn3hfd`/`8d168c501b0d` 为 `OOMKilled=true、exit=137`，Swarm 创建替代 task `yhl8m7kgd5qg`；node1/node2 全程 200，node3 仅短暂 502/503，并在约 11 秒恢复 200。正式 PostgreSQL/Redis/Caddy storage 和 TLS 身份随后复核不变；该方法验证 cgroup 硬限制与单副本恢复，不把外部压力等同于所有生图 limiter 的容量压测；
- migration 故障使用临时逻辑数据库、临时版本化 Secret 与单副本临时 Swarm service，正式配置内容只在管道中把 `database.dbname` 替换为 `sub2api_g4b2c`，不打印或落盘应用 Secret。临时库只预置 `001_init.sql` 的错误 checksum；当前固定镜像 task `7j9f05g93ub6` 明确以 exit 1 失败，日志命中 `migration 001_init.sql checksum mismatch`，没有 server started/listening/ready 记录；
- 临时库只保留 1 条故意错误记录且未创建 `users`，正式库前后均为 `236/236/0/0`。临时 service、Secret 和数据库最终均确认不存在。此前两个直接 `docker run` 尝试分别因缺少等价可写数据目录或未挂载 Swarm Secret 而在 migration 前退出，均已清理且不计有效验证。

禁止在本阶段执行：删除持久化卷、`docker stack rm` 通用卸载、破坏真实业务数据、生产 DNS 修改或未记录的 `--force` 删除。

### 9.5 S4-E：本地容量与稳定性记录（部分完成，生产定标延期）

- [ ] 分别记录单/双/三副本普通请求、生图和长连接基线；
- [ ] 已采集单副本 OOM 的 `memory.current`、`memory.peak` 与 cgroup 事件；Go heap/GC、业务请求/响应字节、网络和磁盘的完整容量基线延期到生产前专项；
- [ ] 记录最热副本、DNS/入口分布偏差模拟和共享 PostgreSQL/Redis 瓶颈；
- [ ] 验证 4G 本地资源档的 reservation/limit 和有界拒绝语义；
- [ ] 明确报告本地数据不能推导生产配额或 200M 聚合带宽效果。

### 9.6 阶段 4 退出门槛

- [x] 最小环境恢复后 Sub2API/Caddy 已回到 `3/3` 且每节点最多一个，PostgreSQL/Redis 为 `1/1`；
- [x] 当前本地实施基线内的多实例安全专项全部通过；
- [ ] shared TLS storage 的 Caddy 重启恢复及 Redis 短时中断期间既有证书服务已通过；Redis 不可用时 Caddy 冷启动与续期协调作为已知限制延期；
- [x] 滚动更新、失败暂停和旧组合回滚可复现；
- [x] 已授权故障矩阵全部完成：低风险子集、Redis/PostgreSQL 暂停恢复、两个数据节点、单副本 OOM 和受控 migration 失败均通过；node1/PostgreSQL 首次失败历史保留，最小修正后的复测通过；
- [x] 未把本地验证表述为生产 HA、容量、DNS 摘除或灾难恢复证明。

## 10. 阶段 5：环境交付

### 10.1 交付物

- [x] 更新后的最终部署清单与平台 digest 清单；
- [x] `deploy/cluster` Stack、Caddy、环境模板、Taskfile 和必要脚本清单；
- [x] GoTask 发布、验证、回滚、日志和节点命令清单；首期人工 drain/undrain 操作保留在手册，不建设自动化任务；
- [x] Config 名称/内容摘要、Secret 名称/object ID 与镜像 digest 的发布对应表；
- [x] 单次 bootstrap、migration、`*_notx.sql` 恢复和 forward-only 边界手册；
- [x] Caddy TLS storage、本地 CA、证书协调、重启和恢复手册；
- [x] ext 风险/修补/测试映射和原项目差异清单；
- [x] fork/upstream 同步记录、版本组合和发布追溯记录；
- [x] 本地多实例验收报告与已授权故障矩阵；完整容量/热点/稳定性定标明确延期到生产前专项；
- [x] 已知限制、遗留风险和 worker 扩容条件；
- [x] DNSPod 多 A、公网 ACME、生产域名和切流的后续设计项；
- [x] “当前不处理 DNS 故障节点摘除”的风险接受记录。

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

`G5` 交付确认（2026-07-27）：**本地实施基线已通过并关闭**。最终运行态为三个 manager Ready/Reachable、唯一 Leader，Sub2API/Caddy `3/3`、PostgreSQL/Redis `1/1`，三个 HTTPS `/ready=200`；Sub2API restart condition 为 `any`，其他 service 保持 `on-failure`，固定 `ext.3` 镜像、Config/Secret、placement、volume 与数据身份均通过最终校验。此确认接受上列边界，并把完整容量定标、Redis 不可用时 Caddy 冷启动/续期、生产 AMD64 部署、DNSPod、真实迁移、备份恢复及生产可观测性留作后续独立方案，不以未执行项伪装为已验证。

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
- [x] `G4-B2a/S4-D` 低风险故障子集已执行并闭环；
- [x] `G4-B2b-1` Redis 暂停/恢复已执行并闭环；
- [x] `G4-B2b-2a` PostgreSQL readiness 失败已按 `backend/extends/lifecycle` 两文件白名单完成最小修补与测试；
- [x] 修补候选的 `ext.3` 版本提升与本地 ARM64 构建已独立授权并闭环；
- [x] 候选节点分发、活动清单变更、受控部署和原场景现场复测已另行授权并闭环；
- [x] `G4-B2b-2b` 数据节点故障执行前只读审查已完成并拆分两个实际授权；
- [x] `G4-B2b-2b-1` node2/Redis 数据节点停止/恢复已执行并闭环；
- [x] `G4-B2b-2b-2` node1/PostgreSQL 首次执行未通过的历史证据已保留；最小静态修正、运行态应用与同场景复测已分别完成并通过；
- [x] `G4-B2b-2b-2-recovery` 最小环境恢复已完成并通过；
- [x] `G4-B2b-2b-2-fix-review` 配置层只读复盘已完成并通过；
- [x] `G4-B2b-2b-2-fix`、`fix-deploy`、`retest` 与 `G4-B2c` 已分别取得本轮一次性授权并完成；
- [x] 阶段 5 交付物足以关闭本地实施基线，延期项和不得外推的结论已明确；
- [x] `G1/G2/G3/G4/G5` 当前本地实施基线均已完成；生产、DNS、真实迁移、容量定标和灾难恢复仍需独立授权。

当前 G0 至 G5 的本地实施基线已全部关闭。活动镜像仍为 `ext.3`，annotated tag `v0.1.165-ext.3` 固定到 `6c859d2d83e03c49fb49a53e530932d7a6c789d7`；最终集群为 Sub2API/Caddy `3/3`、PostgreSQL/Redis `1/1`，三个入口与 `release:verify` 通过。node1/PostgreSQL 首次失败、无效采样尝试和未触发 cgroup 的首个 OOM 注入均保留为历史证据，不冒充成功；后续有效复测与故障注入已通过。下一阶段只能在新的授权下进入生产容量/可观测性、Redis/Caddy 剩余边界、AMD64 生产部署、DNSPod 或真实数据迁移。
