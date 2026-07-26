# Sub2API 多节点部署实施计划

> 状态：第三轮收敛意见已采纳，待最终审核；尚未授权实施
> 创建日期：2026-07-26
> 适用范围：三个 Multipass ARM64 节点的本地 Docker Swarm 验证，以及 AMD64 生产制品与配置基线
> 方案来源：[`Sub2API-MultiNode-Deployment.md`](./Sub2API-MultiNode-Deployment.md)
> 运维契约：[`GoTask-runbook.md`](./GoTask-runbook.md)
> 节点事实：[`Multipass-Nodes.md`](./Multipass-Nodes.md)

## 1. 目的与当前边界

本文把已经完成的多节点部署方案拆解为可执行、可验证、可停止和可回滚的实施步骤。阶段编号严格沿用方案文档第 7 节：阶段 0 至阶段 5。

当前只创建实施计划，不执行下列操作：

- 不修改 Sub2API 源码、发布 workflow 或 GoReleaser 配置；
- 不创建 `backend/extends`、`backend/extends/VERSION` 或 `deploy/cluster`；
- 不构建、推送或覆盖 GHCR 镜像/tag；
- 不安装 Docker/GoTask，不初始化或修改 Swarm；
- 不创建 Docker Secret/Config、网络、service、volume 或数据库；
- 不登录节点、不执行 bootstrap、migration、故障注入或生产切流；
- 不配置 DNSPod，不处理 DNS 故障节点摘除；
- 不提交或推送本计划，除非另行授权。

批准本文只代表认可实施顺序和门槛，不自动授权任何源码、外部制品、节点或生产变更。

## 2. 实施原则

1. **最小改造**：先使用共享 PostgreSQL/Redis、Swarm、Caddy、Secret/Config 和资源限制解决问题；只有无法由部署消除的多实例安全风险才进入 `backend/extends`。
2. **新增文件优先**：ext 实现及其实现测试优先新增到 `backend/extends`；原项目已有文件只保留必要的 Wire/router/参数等薄接入点，并逐项登记原因和修改范围。涉及原包私有行为的回归测试允许就地新增，不为目录合规导出 API 或增加包装层。
3. **若无必要勿增实体**：默认不新增 Ent/domain 实体、数据库表、leader 服务、配置中心、通用插件机制或调度框架。
4. **扩展默认启用**：确认需要的 ext 修补不设置功能开关；三个副本运行相同版本和行为。
5. **部署与代码分离**：`backend/extends` 只存代码修补；`deploy/cluster` 只存集群配置、GoTask 入口和必要短脚本。
6. **不可变发布**：镜像、Config、Secret 均不原地覆盖；部署使用平台镜像 digest，应用更新只通过 Swarm 完成。
7. **本地先行**：先完成 ARM64 本地验证；AMD64 只形成制品与生产配置基线，不在本计划中执行生产部署或切流。
8. **证据先于结论**：每个阶段必须保存版本、digest、对象引用、节点/task 状态、日志和验收结果，不能只以 `/health=200` 或 `docker service ls` 判定成功。
9. **阶段门槛不可跳过**：前一阶段未通过或缺少对应授权时，不进入后一阶段。
10. **人工上游同步**：不设置自动同步频率；发生冲突时人工处理，不执行共享分支 rebase/force-push。

## 3. 当前实施基线

以下状态是制定本计划时的仓库快照，正式实施前必须重新核对：

| 项目 | 当前状态 |
| --- | --- |
| fork 分支 | `main`，跟踪 `origin/main` |
| 文档基线 commit | `36ff73e93d4850fcfacd83bd826f192d4b32cc59` |
| upstream VERSION | `backend/cmd/server/VERSION = 0.1.165` |
| fork VERSION | `backend/extends/VERSION` 尚不存在；计划首值为 `ext.1`，实施前仍须核对历史 fork tag |
| 组合版本 | 计划为 `0.1.165-ext.1`，tag/镜像 tag 为 `v0.1.165-ext.1` |
| `backend/extends` | 尚不存在 |
| `deploy/cluster` | 尚不存在 |
| release workflow | 仍会改写/同步 `backend/cmd/server/VERSION`，不符合已确认的双 VERSION 只读组合规则 |
| GoReleaser | 两份配置仍为 `prerelease: auto`，尚未显式注入完整 fork `main.Version` |
| 本地节点 | `node1`、`node2`、`node3`，均为 Ubuntu ARM64、2 vCPU、4G 内存、20G 磁盘 |
| Swarm/业务 service | 本计划未确认当前实时状态；阶段 1 必须重新取证，不从文档推断 |

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
| `G0` 计划审核 | 只审核本文，不修改实施文件或环境 | 待审核 |
| `G1` 仓库实施授权 | 允许修改版本/发布文件，创建 `backend/extends`、`deploy/cluster` 和测试；不推送镜像、不修改节点 | 未授权 |
| `G2` 制品发布授权 | 允许向私有 GHCR 推送新的不可变 tag/manifest，并记录平台 digest | 未授权 |
| `G3` 本地环境实施授权 | 允许安装/配置 Docker、初始化 Swarm、创建 Secret/Config/service/volume，并在三个 Multipass 节点部署 | 未授权 |
| `G4` 故障演练授权 | 允许在本地测试环境执行 task kill、节点停止、依赖中断、OOM 和受控 migration 失败测试 | 未授权 |
| `G5` 交付确认 | 确认本地验收结论并关闭实施计划 | 未授权 |

任何授权都只覆盖表中动作。`G1` 不隐含 `G2/G3`，`G3` 不隐含 `G4`，本地完成不隐含生产授权。

### 4.2 总体阶段状态

| 阶段 | 状态 | 进入条件 | 退出条件 |
| --- | --- | --- | --- |
| 0. 需求冻结与架构决策 | 已完成 | 方案设计审核 | 本地设计项确认、实施计划形成 |
| 1. 节点与基础设施基线 | 未开始 | `G1`；涉及 GHCR/节点时再分别取得 `G2/G3` | 发布链、制品、配置骨架和三 manager 基线通过 |
| 2. 数据服务与单副本基线 | 未开始 | 阶段 1 通过且已取得 `G3` | PostgreSQL/Redis、单次 bootstrap、单副本与本机 Caddy 基线通过 |
| 3. 多实例前置收敛 | 未开始 | 阶段 2 通过且代码修补范围再次确认 | 必要 P0 修补、进程级测试和静态验证满足门槛；不启用 `node2`/`node3` 应用副本 |
| 4. 三副本与故障演练 | 未开始 | 阶段 3 通过且已取得 `G4` | 三副本、TLS、滚动更新、回滚和故障矩阵通过 |
| 5. 环境交付 | 未开始 | 阶段 4 通过 | 交付物、限制和验收报告完成并取得 `G5` |

## 5. 阶段 0：需求冻结与架构决策

### 5.1 当前结论

阶段 0 已由方案文档完成。本文不重新讨论已确认架构，只在实施开始前做一致性复核。

### 5.2 实施前复核清单

- [ ] 确认方案文档状态仍为“阶段 0 门槛已满足”；
- [ ] 确认本计划已人工审核；
- [ ] 确认三个现有节点均为 `manager + worker`，后续容量节点只作为 worker；
- [ ] 确认 PostgreSQL 固定 `node1`、Redis 固定 `node2`，不做数据服务 HA；
- [ ] 确认 Caddy 每节点只代理本机 Sub2API，不改用 Traefik/routing mesh；
- [ ] 确认本地使用 `sub2api.test`、`tls internal`，不接 DNSPod；
- [ ] 确认生产容量与监控细项继续延期，不作为本地阻塞项；
- [ ] 明确本轮获得的是 `G1`、`G2`、`G3` 中哪一层授权。

### 5.3 停止条件

- 方案与计划存在未解决冲突；
- 实施范围要求新增未批准实体、控制面或通用扩展框架；
- 用户仅批准文档而未批准对应实施门槛。

## 6. 阶段 1：节点与基础设施基线

阶段 1 分为仓库侧准备、制品发布和节点侧基线三部分。三部分分别受 `G1/G2/G3` 控制，不因同属一个阶段而合并授权。

### 6.1 S1-A：仓库与上游基线复核

- [ ] 确认工作树范围，保留用户无关改动；不使用 `git add -A` 暗中纳入其他文件；
- [ ] 核对 `origin`/`upstream` URL、当前分支和远端默认分支；
- [ ] fetch `origin` 和 `upstream`，只读比较差异；没有人工明确要求时不合并 upstream；
- [ ] 读取当前 `backend/cmd/server/VERSION`，查询历史 fork tag，计算下一个未使用的 `ext.N`；
- [ ] 记录 upstream commit、fork commit、Go/Docker/GoTask 版本和实施日期；
- [ ] 运行实施前后端基线测试，区分既有失败与本轮引入失败。

产物：版本输入表、remote/branch 记录、基线测试记录。

### 6.2 S1-B：双 VERSION 与发布链最小改造

在 `G1` 授权后按独立提交实施：

- [ ] 新增 `backend/extends/VERSION`，首个实际值由历史 tag 核验决定；当前计划值为 `ext.1`；
- [ ] 保持 `backend/cmd/server/VERSION` 不变；增加测试或 CI 检查，阻止 fork 发布流程改写该文件；
- [ ] 修改 `.github/workflows/release.yml`，只读组合 upstream VERSION 与 ext VERSION；
- [ ] 校验格式分别为 `X.Y.Z` 与 `ext.N`，并校验触发 tag 严格等于 `v${FORK_VERSION}`；
- [ ] 删除 workflow 中写入、上传替换或自动提交 `backend/cmd/server/VERSION` 的步骤；
- [ ] 修改 `.goreleaser.yaml` 和 `.goreleaser.simple.yaml`，通过现有 `ldflags` 注入不带前导 `v` 的完整 `main.Version`，继续注入 `Commit`、`Date`、`BuildType`；
- [ ] 将 GitHub Release 明确设置为 `prerelease: false`；
- [ ] 保留 ARM64/AMD64 架构 tag 和 multi-arch manifest，确保 tag 带前导 `v`；
- [ ] 增加只读校验：两个 VERSION 文件未被 CI 修改、tag/运行时版本一致、历史 ext 序号未复用。

验证：

- [ ] `git diff` 证明 upstream VERSION 未变化；
- [ ] GoReleaser 配置校验通过；
- [ ] 本地或 CI dry-run 得到 `FORK_VERSION=0.1.165-ext.1`（若基线未变化）；
- [ ] 构建二进制返回完整 fork `Version/Commit/Date`；
- [ ] 没有为 `-ext.N` 修改应用内更新逻辑。

### 6.3 S1-C：Caddy 与双架构制品输入

- [ ] 使用固定的 Caddy `v2.11.4` 和 `github.com/pberkel/caddy-storage-redis@v1.8.1`；
- [ ] 以单一最小 Dockerfile 作为 Caddy 自定义镜像构建输入，不建立额外镜像框架；
- [ ] Caddy 使用独立构建 job/package，不进入 Sub2API 镜像；
- [ ] 分别构建 `linux/arm64`、`linux/amd64` 子镜像和 multi-arch manifest；
- [ ] Sub2API 复用修正后的现有 GoReleaser multi-arch 发布链；
- [ ] PostgreSQL `18-alpine`、Redis `8-alpine` 在实施时固定对应平台 digest；
- [ ] 保存构建输入、源码 revision、模块清单和平台 digest；SBOM/扫描仅在现有发布链可直接产出时保留，新增扫描工具延期到生产准入。

只有取得 `G2` 后才允许推送以下私有 package：

- `ghcr.io/ryanpenn/sub2api`
- `ghcr.io/ryanpenn/sub2api-caddy`

验证：

- [ ] `docker buildx imagetools inspect` 显示 ARM64/AMD64 平台正确；
- [ ] `caddy version` 为固定版本；
- [ ] `caddy list-modules` 包含 `caddy.storage.redis`；
- [ ] 部署记录使用平台子镜像 digest，而不是只记录 manifest tag；
- [ ] GHCR 不可达时停止，不回退到 Docker Hub、`latest` 或本地可变 tag。

### 6.4 S1-D：`deploy/cluster` 最小骨架

在 `G1` 授权后创建下列最小结构；不创建空控制器、占位框架或长期运行组件：

```text
deploy/cluster/
├── Taskfile.yml
├── taskfiles/
│   ├── validate.yml
│   ├── release.yml
│   └── ops.yml
├── stacks/
└── env/
    ├── local-arm64/
    └── production-amd64/
```

- [ ] 根 Taskfile 只组合 `validate/release/ops` 命名空间；
- [ ] `validate` 覆盖 Docker Context、Manager/quorum、架构、label、placement、资源限制、固定 digest 和 Config/Secret 引用；
- [ ] `release` 先实现 `plan/apply/verify/rollback/bootstrap` 契约，不提供通用 `uninstall`；
- [ ] `ops` 第一期只提供状态、日志和节点检查；`drain/undrain` 自动化延期，需要时先按手册人工执行；
- [ ] ARM64/AMD64 环境文件只保存非敏感参数和脱敏模板；
- [ ] Stack 不包含明文 Secret、可变 tag、Docker Socket 或未受控 bind mount；
- [ ] Caddyfile 包含本机 upstream、Redis storage、回环 admin API、内部 TLS 和管理端更新接口阻断规则；
- [ ] 不预建空 `scripts/` 或占位文件；复杂逻辑只有在 Taskfile 无法安全表达时才按需新增短脚本。

阶段 1 只要求配置可渲染、可静态校验；没有 `G3` 时不得执行 `release:apply`。

### 6.5 S1-E：节点与 Swarm 基线

取得 `G3` 后执行：

- [ ] 重新核对三个节点的 IP、主机名、Ubuntu/ARM64、CPU、内存、磁盘和当前 service；
- [ ] 核对时间同步、DNS、内核参数、端口占用、磁盘空间和 Docker 日志轮转；
- [ ] 固定 Docker/GoTask 版本及安装来源，保存 checksum；
- [ ] 由 `node1` 初始化或确认 Swarm，`node2`/`node3` 以 manager 加入；三个 manager 均保留 worker 能力；
- [ ] 验证 manager 数量为 3、quorum 正常；普通容量扩展不增加 manager；
- [ ] 创建内部 overlay network；Caddy 继续使用 host network，不发布 routing mesh 入口；
- [ ] 设置 `postgres=true` 仅在 `node1`、`redis=true` 仅在 `node2`；
- [ ] 阶段 2 初始仅使 `node1` 具备 `sub2api=true`/`caddy=true`，`node2`/`node3` 完成能力检查但暂不加应用入口 label；阶段 4 再启用，以保持单副本基线；
- [ ] 指定一个 Manager 作为本地人工发布入口，使用 GHCR 只读凭据和 `--with-registry-auth`；
- [ ] 验证没有节点持久化 `write:packages` 凭据。

### 6.6 阶段 1 退出门槛

- [ ] 版本组合、tag、runtime version 和双架构制品可追溯；
- [ ] ARM64 平台 digest 已回填本地环境；AMD64 digest 已记录但未部署；
- [ ] `deploy/cluster` 静态校验通过且未包含 Secret；
- [ ] 三个 manager quorum 正常，数据节点 label 唯一；
- [ ] 本阶段没有部署 PostgreSQL、Redis、Caddy 或 Sub2API 业务 service；
- [ ] 阶段 1 差异按版本发布、集群骨架、节点证据分开审核。

## 7. 阶段 2：数据服务与单副本基线

### 7.1 S2-A：部署共享数据服务

- [ ] 以固定 ARM64 digest 部署 PostgreSQL 单实例，placement 绑定 `node1` 和明确的本地持久化目录；
- [ ] 以固定 ARM64 digest 部署 Redis 单实例，placement 绑定 `node2` 和明确的本地持久化目录；
- [ ] 禁止两项 service 漂移到无原数据目录的节点；
- [ ] PostgreSQL/Redis 不对测试入口公开，只通过内部网络或已确认私网端点访问；
- [ ] Redis 启用本地 AOF/RDB 基础持久化；不把它表述为跨节点备份；
- [ ] S3 配置保持为空，定时 S3 备份保持禁用；
- [ ] 应用数据库账号、Redis ACL 和 Caddy TLS storage ACL 按消费者边界最小授权。

验证：健康检查、持久化目录、placement、重启后加载、网络暴露和资源限制均符合方案。

### 7.2 S2-B：创建 Config/Secret

- [ ] Config 使用 `sub2api-{env}-{purpose}-{sha12}`；
- [ ] Secret 使用 `sub2api-{env}-{purpose}-vNNN`，不记录 Secret 内容摘要；
- [ ] 创建本地专用 `app-config`、PostgreSQL、Redis app、Redis Caddy、Caddy storage key 等 Secret；不复用生产值；
- [ ] JWT/TOTP 默认收敛在 `app-config`，只有消费范围确实不同时才拆分；
- [ ] 创建经审计的 `model_pricing.json` Config，并固定匹配的不可变远程 URL/hash；
- [ ] 创建 Caddyfile Config；发布记录保存 Config 名称/内容摘要及 Secret 名称/object ID；
- [ ] 所有明文值通过受控输入创建，不进入 Git、命令行参数、Stack、Config、镜像或日志；
- [ ] 本地 Secret 丢失时按方案重建环境，不额外建设 Secret 备份系统。

### 7.3 S2-C：单次 bootstrap

- [ ] 创建一次性 bootstrap 管理员密码 Secret；
- [ ] 只启动一个临时受控实例，显式提供管理员密码、JWT/TOTP Secret 并设置 `AUTO_SETUP=true`；
- [ ] 等待 migration、schema checksum、管理员和必要 seed 完成；
- [ ] 验证 bootstrap 成功后关闭并删除临时实例；
- [ ] 删除一次性管理员密码 Secret；
- [ ] 正式 service 固定 `AUTO_SETUP=false`，只读挂载权威 `app-config`；
- [ ] 不保留 bootstrap service，不新增 migration Job 或协调实体。

### 7.4 S2-D：单副本与本机 Caddy

- [ ] 使用 `global` service，但阶段 2 只有 `node1` 带 `sub2api=true`/`caddy=true`，因此各运行一个 task；
- [ ] Sub2API 只发布供本机 Caddy 使用的本机端口，不对测试入口直接暴露；
- [ ] Caddy 使用 host network 绑定 `80/443`，admin API 仅监听 `127.0.0.1:2019`；
- [ ] Caddy 固定代理本机 Sub2API，不通过 routing mesh；
- [ ] Caddy 使用 Redis storage 的专用 ACL、DB、key prefix 和 encryption key；
- [ ] 通过部署配置统一启用现有每副本图片 limiter，并为三个副本预设相同参数；具体数值以 4G 本地资源基线为输入，不解释为生产配额；
- [ ] 本地使用 `sub2api.test` 与 `tls internal`；
- [ ] 通过 `curl --noproxy '*' --resolve` 验证 TLS、`/health`、未来 `/ready` 基线及核心 API；
- [ ] 记录普通 HTTP、SSE、WebSocket 和生图的单副本资源基线。

### 7.5 阶段 2 退出门槛

- [ ] PostgreSQL 只在 `node1`、Redis 只在 `node2`，重启不漂移到空目录；
- [ ] bootstrap 只执行一次，一次性密码 Secret 已删除；
- [ ] 正式实例 `AUTO_SETUP=false`，权威 Config/Secret 引用可追溯；
- [ ] 单副本经本机 Caddy 正常访问，Sub2API 端口不能绕过入口策略；
- [ ] Caddy Redis storage 重启后仍能读取本地 CA/证书数据；
- [ ] 4G 本地资源限制生效，但没有把结果解释为生产容量。

## 8. 阶段 3：多实例前置收敛

阶段 3 先用测试证明风险，再做最小修补。不得把方案中的“可能风险”直接转化为新功能。本阶段保持阶段 2 的单副本 Swarm 基线，不给 `node2`/`node3` 添加应用 label；并发行为通过目标单元测试、进程级集成测试和协议级 stub/mock 验证，真实三节点测试统一在阶段 4 执行。

### 8.1 S3-A：多实例风险与接入点盘点

- [ ] 重新沿当前源码确认 OAuth SessionStore、并发槽清理、图片 limiter、readiness/drain、WebSocket、本地文件和后台任务调用链；
- [ ] 为每项建立“风险证据 → 最小修补 → 测试 → 目录例外”映射；
- [ ] 确认能由部署解决的项目不进入 `extends`；
- [ ] 确认是否需要既有目录中的 Wire/router/参数薄接入点，并记录修改行范围；
- [ ] 实施前按第 8.8 节白名单形成精确文件清单；未列入的 upstream 文件默认不得修改；
- [ ] 如发现必须新增实体/表/通用抽象，立即停止并重新评审，不直接实施。

### 8.2 S3-B：P0 代码修补

按独立、小范围提交实施，每项先补失败测试：

1. **并发槽清理**
   - [ ] 先用原包回归测试复现：新副本启动会清除其他健康进程前缀的槽位或共享等待计数；
   - [ ] 最小修补仅删除 `backend/internal/service/wire.go` 中启动时调用 `CleanupStaleProcessSlots` 的路径；
   - [ ] 新副本启动不删除其他健康 request prefix 的槽位或共享等待计数；
   - [ ] 继续复用现有 score/TTL/活跃索引和周期清理，不改写既有回收模型；
   - [ ] 不创建 `extends/concurrency` 包装层，不新增 owner、heartbeat、接口或持久化模型；
   - [ ] 该项登记为“一处 upstream 原文件直接修补 + 原包 test-only 回归测试”例外。
2. **OAuth Redis SessionStore**
   - [ ] `backend/extends/oauthsession` 只提供一个可复用的 typed Redis JSON store，统一 provider namespace、TTL 和错误语义；
   - [ ] 一次性消费使用 Lua 在单次 Redis 操作中完成“读取、比较预期 state、匹配后删除并返回”；state 不匹配不得删除，Redis 错误必须 fail closed；
   - [ ] 五个 OAuth service 各自定义最窄 typed interface，只修改 store 字段、构造注入及 `Put/Take` 调用，不直接 import `extends`；
   - [ ] `backend/extends/wire.go` 集中创建 typed store，`cmd/server/wire.go` 只接入一个 `extends.ProviderSet`；
   - [ ] 保持五个 `internal/pkg/*/oauth.go` 中的上游内存实现不变，不依赖粘性会话，不回退进程内 store，不新增数据库实体。

### 8.3 S3-C：最小 readiness、排空与 WebSocket

- [ ] 保留 `/health` 作为 liveness；新增 `/ready` 表达启动、必要共享依赖和 draining 状态；依赖检查只复用现有客户端做短超时按需探测，不新增后台监控框架；
- [ ] `SIGTERM` 先置 draining、拒绝新请求，再按配置窗口排空；窗口不大于 Swarm `stop_grace_period`；
- [ ] HTTP/SSE 使用服务器 shutdown 语义完成排空；
- [ ] WebSocket 只增加最小客户端连接 registry：draining 后拒绝新 upgrade，已有连接可继续到排空窗口结束，到期发送 `1012 Service Restart` 后关闭；
- [ ] 第一期不识别“当前 turn/新 turn”，不侵入 forwarding loop；精确的“当前 turn 完成、新 turn 拒绝”作为 P1 延期并需单独评审；
- [ ] 既有 `response_id -> conn_id`、`session -> conn_id` 和 turn state 保持原实现，不复制到 lifecycle/registry；重连不续接其他副本未完成 turn；
- [ ] 不为 draining/连接 registry 新增 Redis key、数据库实体或跨副本协调状态。

### 8.4 S3-D：图片内存保护证据门槛

- [ ] 通过部署配置启用现有每副本 limiter，并确认三个环境模板使用相同参数；不新增 ext 开关、Redis 集群总计数或 limiter 框架；
- [ ] 先确认同步 Responses、Images 和复用 Images/GrokImages 的异步路径已进入现有 limiter，不重复接入；
- [ ] Batch worker 保持现有单 worker/副本和跨实例 job lock，不强行接入 HTTP limiter；
- [ ] 重点测试 WebSocket 生图、Gemini native 及本阶段实际启用的其他高内存入口；只有测试证明某一具体入口绕过保护时，才在该调用点增加最小接入；
- [ ] 未启用或没有高内存证据的路径只记录盘点结论，不产生代码提交。

### 8.5 S3-E：后台任务与启动副作用

- [ ] 列出所有实际启用的周期任务、claim 方式、外部副作用和并发语义；
- [ ] Account expiry、Proxy expiry 先验证既有条件更新/事务语义；测试通过即保持不加锁、不改代码；
- [ ] Scheduled Test 作为当前唯一明确候选，先证明多进程会重复执行外部测试，再决定是否增加门控；
- [ ] 经测试证明必须单次执行的任务只复用现有 Redis leader lock，并验证 PostgreSQL advisory lock 回退；
- [ ] Redis/PostgreSQL 均不可协调时跳过当次，不无锁执行；
- [ ] S3 定时备份继续禁用并验证零执行；
- [ ] 不创建 `extends/scheduler`、leader facade、任务注册表或通用调度框架；没有失败证据时不产生后台任务代码提交。

### 8.6 S3-F：migration 与配置一致性测试

- [ ] 使用三个测试进程或并发数据库 session 模拟同时启动，确认只有一个 session 执行 migration SQL；本阶段不启用三个 Swarm 副本；
- [ ] 等待副本获锁后重新核对 `schema_migrations`/checksum，不重复执行；
- [ ] 对全新数据库执行三次冷启动，记录锁等待、SQL 和总耗时；最慢不得超过 5 分钟；
- [ ] 验证事务 migration 失败回滚和 10 分钟总上下文超时不进入 ready；
- [ ] 在一次性测试数据库中验证 `*_notx.sql` 无效索引检查、受控清理和相同 digest 重试；不修改 migration 记录、不自动删除业务数据；
- [ ] 对 schema 变化标记 `backward-compatible` 或 `forward-only`；
- [ ] 验证 JWT bootstrap、Simple Mode 默认分组和管理员并发 seed 的幂等/唯一约束；
- [ ] 静态验证三个目标副本在渲染后的 Stack 中引用同一 `app-config` Secret 名称；object ID 与跨节点 JWT/TOTP 行为留到阶段 4 实机确认；
- [ ] 验证模型价格 Config 名称/摘要、`local_hash`、不可变 URL/hash 和旧版本回滚。

### 8.7 S3-G：部署层安全规则与进程级测试

- [ ] Caddy 阻断在线更新检查、可回滚版本查询、原地更新和原地回滚接口；
- [ ] 只保留 `/api/v1/admin/system/version`，返回完整 fork 版本；
- [ ] 验证没有发布的 Sub2API 端口可绕过 Caddy；
- [ ] 完成目标单元测试、进程级并发测试及 OAuth/WebSocket 协议级 stub/mock；真实双/三副本 HTTP、SSE、WebSocket、OAuth、limiter 和后台任务测试留到阶段 4；
- [ ] 完成 Caddy shared Redis storage 的证书协调、锁、重启和恢复测试准备；
- [ ] 完成 `task validate:*`、`release:plan` 和静态 Stack 渲染；
- [ ] 不把人工上游同步演练作为本地退出门槛；同步仍按 fork 维护规则由人工按需发起。

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
- 五个 OAuth service 文件：只替换 store 字段、构造注入和 `Put/Take` 调用；
- `backend/internal/service/wire.go`：删除破坏性并发槽启动清理；仅当失败测试成立时，为 Scheduled Test 增加既有 leader-lock 注入；
- 图片 handler：仅在测试证明遗漏的具体高内存入口增加现有 limiter 调用；
- 对应原包测试文件：允许验证私有行为，统一登记为 test-only 例外。

明确禁止：

- 修改 Ent schema、domain 实体或新增 migration；
- 新建通用 scheduler、plugin、leader facade、connection-state 或 limiter 框架；
- 新增跨节点 WebSocket 状态；
- 为把测试放进 `extends` 而导出私有 API、增加 wrapper 或重复 adapter；
- 修改五个 `internal/pkg/*/oauth.go` 内存 SessionStore，或为 fork 修改应用内更新逻辑。

### 8.9 阶段 3 验证与退出门槛

- [ ] `gofmt`、目标单元/集成测试及 `cd backend && go test ./...` 通过；
- [ ] Stack/Caddy/Taskfile 静态校验通过；
- [ ] ext 实现及其实现测试位于 `backend/extends`；涉及原包私有行为或薄接入点的回归测试允许就地新增，并登记为 test-only 例外；
- [ ] 实际修改文件全部位于第 8.8 节白名单，且未为了目录合规导出私有 API 或增加包装层；
- [ ] 未增加无关功能、开关、实体、表或通用框架；
- [ ] 每项修补都有失败证据、修补测试和回归测试；
- [ ] 没有失败证据的图片入口和后台任务没有产生代码提交；
- [ ] 版本、fork commit、Config/Secret 和 schema 兼容性可追溯；需要部署验证的镜像 digest 在取得 `G2` 后记录；
- [ ] 形成“不改代码基线”与“必要最小改造”的差异清单并通过人工审核。

## 9. 阶段 4：三副本与故障演练

阶段 4 需要 `G4`。所有故障注入只作用于本地测试环境，不触碰生产或已有生产数据。

### 9.1 S4-A：启用三个 global 副本

- [ ] 确认阶段 3 制品、Config/Secret 和回滚目标已固定；
- [ ] 给 `node2`、`node3` 添加 `sub2api=true`/`caddy=true`；
- [ ] 验证 Sub2API/Caddy `global` service 在每个合格节点恰好一个 task；
- [ ] 验证每个 Caddy 固定代理本机 Sub2API，未经过 routing mesh；
- [ ] 使用同一 Local CA，通过 `curl --noproxy '*' --resolve` 分别访问三个节点；
- [ ] 核对三个 Caddy 的证书 subject、serial、指纹和 Caddyfile Config；
- [ ] 核对三个 Sub2API 的镜像 digest、完整版本、配置对象和节点落位。

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

- [ ] 三个 Sub2API/Caddy task 稳定且每节点最多一个；
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

- [ ] 阶段顺序与方案文档第 7 节一致；
- [ ] `G1` 至 `G5` 授权边界清楚；
- [ ] 阶段 1 的版本/发布、制品、配置骨架和节点步骤是否需要调整；
- [ ] 阶段 2 单副本采用“仅 node1 添加应用/入口 label”的过渡方式是否接受；
- [ ] 阶段 3 的 P0/P1 修补顺序和独立提交边界是否接受；
- [ ] 阶段 3 修改文件是否全部落在第 8.8 节白名单，且条件提交没有被预设为必做；
- [ ] 阶段 4 故障注入范围是否接受；
- [ ] 阶段 5 交付物是否足够；
- [ ] 是否仅授权 `G1`，或同时授权 `G2/G3`。

审核通过前，本计划保持“待人工审核，尚未授权实施”状态。
