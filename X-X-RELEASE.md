# X-X 自维护版本与发布

本文件只说明自维护版本、更新提示和打包约定；生产升级继续遵循 x-x 仓库的 `AI-Gateway-NewAPI-Operations.md` 蓝绿部署流程。

## 版本来源

- 唯一版本文件：`common/x_x_version.txt`，当前源码修订为 `v1.0.0-rc.51+lqy007700+v1`（尚未发布）；发布与生产部署分开执行。
- `v1.0.0-rc.47` 表示合并的官方 QuantumNous/new-api 基线，`+lqy007700+v1` 表示该基线上的第一个自维护发布修订。修订号必须是从 1 开始的正整数，不允许前导零。
- 同一基线再次发布时递增修订号，例如 `v1.0.0-rc.47+lqy007700+v2`；合并新的官方基线后重新从 1 开始，例如 `v1.0.0-rc.48+lqy007700+v1`。
- 新发布标签与完整版本相同，例如 `v1.0.0-rc.47+lqy007700+v1`，不添加 `x-x-custom-` 前缀。历史 `-x-x.N` 版本及 `x-x-custom-` 标签仍可解析、比较和打开；不修改、删除或重新发布历史标签。
- 官方 `VERSION` 及来源归属保留。程序通过 `/api/status` 的新增字段 `x_x_build` 提供内嵌版本、官方基线、构建提交；旧 `version` 字段及环境变量行为不变。
- 管理员左上角显示 `X-X rc.47 · v1`，悬浮提示和更新弹窗显示完整版本。系统维护页也优先显示程序内嵌的 X-X 版本。普通用户不显示更新入口。
- Docker 构建用 `X_X_BUILD_COMMIT` 写入完整提交号；本地 Go 构建缺省读取 Go VCS 元数据，无法获取时省略，不伪造提交号。

## 更新提示

- 默认检查 `lqy007700/new-api` 的 GitHub Releases，不再把官方未合并版本当成自维护版本的可部署更新。
- 认可新格式标签及历史 `x-x-custom-...-x-x.N` 标签。版本顺序先比较官方基线，再比较 X-X 数字修订号；同一基线的 `v2 < v10`，更高官方基线的 `v1` 大于旧基线的任意修订号。自维护后缀参与数字排序；普通构建元数据不参与排序。
- 必须同时具备已上传、非空的以下两个附件才提示更新：
  - `new-api-x-x-<完整发布标签>-linux-amd64.tar.gz`
  - 同名 `.tar.gz.sha256` 文件。
- 草稿、仅 Actions artifact、缺失附件、旧日期标签及官方原始标签均不提示。附件与 Release 链接中的 `+` 正确编码为 `%2B`。
- 沿用管理员每小时检查、手动检查、忽略版本、网络失败降级和浏览器端无凭据 GitHub 请求。缓存及忽略记录使用独立的 X-X 键，不复用官方旧提示。
- 弹窗保留官方基线链接、显示构建提交和自维护更新说明。可从官方基线页面进入官方发布列表；官方升级仍须先合并、验证和构建。
- GitHub 请求受网络和匿名 API 限流影响，失败时显示错误；不要把 GitHub Token 写进前端。私有仓库不能直接使用此匿名更新源。
- **检查更新不会执行安装、重启、切流或数据库操作。**

## 后续发布步骤

1. 合并需要的官方版本，保留 X-X 的业务修改并完成测试。
2. 更新 `common/x_x_version.txt`，准确记录官方基线及新的数字修订号。
3. 运行 `bash scripts/x-x-build-metadata.sh`，并完成受影响测试、类型检查、lint 与构建。
4. 提交并推送代码。推送 `main` 只触发现有 amd64 Actions artifact 构建，不会部署，也不会自动发布 Release。
5. 准备发布时，创建并推送与文件一致的标签，例如：

   ```bash
   git tag v1.0.0-rc.47+lqy007700+v1
   git push --atomic origin main refs/tags/v1.0.0-rc.47+lqy007700+v1
   ```

   上述仅为本次版本示例；下次先更新版本文件，勿重复使用旧标签。也可以在 `Release New API amd64 image` 工作流中手动填写已存在且匹配的标签。

6. 工作流校验标签与版本文件一致，以**实际检出的提交号**标记镜像并写入程序。先构建 linux/amd64 镜像，再上传归档和 SHA256 到草稿 Release；附件就绪后才发布。
7. 确认工作流成功、Release 附件完整、镜像为 linux/amd64、提交号正确，再按运维文档蓝绿部署。不要用官方原始镜像替代自维护镜像。
8. 新容器预检 `/api/status` 的 `data.x_x_build`，确认版本、官方基线和提交号符合预期后再切流。若 Compose 仍配置 `VERSION`，建议同步为同一版本；界面不会再被旧环境变量误导。

历史 Actions artifact 和旧 Release 不自动改名、删除或重新发布。新版程序识别新旧格式；已部署的旧版更新检查器需要先升级程序才能识别新标签。自维护标签只触发 amd64 镜像发布，官方二进制发布和多架构 Docker 工作流排除这些标签。镜像继续用提交哈希作为 Docker tag。发布与生产部署分开执行。

## rc.45 合并记录（2026-10-19）

- 官方基线：`v1.0.0-rc.45`（`6cf0d90b5ff27f901697dddd61e0c28aba8efefc`）。本次仅合并、验证和推送代码，不创建标签或 Release，不操作生产服务器。
- 保留 X-X 版本及更新提示、普通用户日志中的映射/响应模型隐藏、现有客户端通用错误处理、兑换记录和每日 API Key 用量统计。
- 冲突按调用链处理：保留普通 API 的 `ToClient*` 通用错误出口；Gemini 错误仍交给统一出口，避免直接返回上游错误；任务插件保留所有权识别逻辑，同时合入官方请求体过大状态判定。官方内部诊断脱敏逻辑继续保留。
- 后续部署前注意：官方新增 `tokens.allow_user_quota_query`，由现有迁移流程添加；旧 Token 默认关闭、新 Token 默认开启。用户通知开关保存在现有设置 JSON 中。需在隔离数据库中验证迁移与回退，不可只凭镜像可启动就切流。
- 官方将未持久化配置时的敏感词检查默认值改为关闭，默认词表为空；已保存配置仍从数据库加载。后续部署前应核对实际保存的检查开关和词表，本次不更改线上敏感词配置。
- 补齐 3 个 X-X 自定义接口的个人访问令牌路由声明：兑换记录、每日用量、用量补算继续仅允许浏览器会话，不新增 PAT 权限范围；对应路由覆盖和鉴权回归测试通过。认证相关检查参考 OWASP Authentication / Session Management Cheat Sheets，未作全量 ASVS 合规认证。
- 本地验证：Go 1.26.9 下根模块 `go test ./...`、relaykit 独立 `GOWORK=off go test ./...` 与 `go build ./...` 通过；前端 75 个测试文件、1228 项测试通过，用量工具另用 `bun test` 通过 4 项；类型检查与生产构建通过。按项目 shadcn-ui 约定复用原组件，未新增替代 UI。
- 验证边界：`go test ./model -run '^TestChannelPriorityDatabaseMatrix$' -v` 的 SQLite 3.50.4 测试通过；本机未配置 `TEST_MYSQL_DSN` / `TEST_POSTGRES_DSN`，相应数据库测试跳过，**尚未完成三数据库迁移兼容矩阵**。本次涉及前端文件的 lint 无错误，改动的两处定价工具代码格式检查通过；全库 lint 仍有 156 个位于未改文件的存量错误，不在同步官方版本时批量改写。

## rc.45-x-x.3（2026-10-27）

- 常规模型 API 收到渠道 HTTP 409 时，对客户端保留 409，统一重新包装为 `type=session_error`、`code=session_unavailable` 和“当前会话已不可用，请新建会话后重试。”，附带本平台请求 ID。Claude 协议使用 `type=session_error` 和相同文案。
- 不透传渠道原始 message、type、code、param 或 metadata；原始错误继续保留在内部诊断对象中。
- 409 不自动重试或换渠道，即使重试范围包含 409、渠道错误码以 `channel:` 开头也不重试；渠道状态码映射不能将收到的 409 改成其他状态。
- 本次发布 linux/amd64 镜像归档及 SHA256，生产部署仍需另行执行。

## rc.47+lqy007700+v1（2026-10-27）

- 完整合并官方 `v1.0.0-rc.47`，固定提交 `e835b1bfc43b565d625537830dbfb19328925134`，未引入官方 main 的未发布代码。
- 保留隐藏倍率（含显式 0 免费、用户专属倍率优先级）、公开倍率与实际倍率分离、普通用户模型隐私、兑换历史、每日 API Key 用量统计、自定义接口权限及自维护更新源。
- 所有客户端错误继续重新包装。渠道 409 使用固定 `session_error / session_unavailable` 和新建会话提示，不透传上游字段、不重试、不换渠道、不被状态映射覆盖；504、524 继续禁止重试。已提交 SSE 响应头时发送脱敏流内错误事件，保持已提交的 HTTP 状态。
- 合入流式完成判定、Claude 保活交接、大事件和写入失败处理；Responses SSE 的 error、response.error、response.failed 及 status=failed 事件整体重建，清除顶层和嵌套的上游字段，脱敏前保留权威用量。race 检查发现的日志轮转计数器竞争改为原子操作。
- 合入缓存计费与统计、普通渠道和插件共同调度、任务重试重新定价及提交前补足预扣；异步任务按最终成功提交时的快照结算和退款。
- 合入官方数据库、插件和安全修复。工具使用隔离目录中的 Go 1.26.10、Bun 1.4.4，前端使用官方锁文件冻结安装。
- 认证相关合并审阅参考 [OWASP ASVS 5.0.0](https://github.com/OWASP/ASVS/tree/v5.0.0)（V6/V7，错误出口参考 v5.0.0-16.5.1）、[Authentication](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)、[Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)、[Forgot Password](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html) Cheat Sheets；重点核查邮件发送失败不暴露凭据、不重试或回退重复投递，以及既有会话、验证码与审计回归。未作全量 ASVS 合规认证。
- 本次只提交、推送和触发 linux/amd64 镜像发布；确认任务已排队或运行即结束，不等待发布完成，不进行生产部署。

### 本次验证

- 后端：Go 1.26.10，根模块、relaykit、tokenkit 的全量 `go test ./...`、`go build ./...`、`go vet ./...` 均通过，三个模块均设置 `GOWORK=off`。
- 流式 race：保活交接、客户端取消、完成与取消竞争、大事件、写入及 flush 失败、Responses 错误事件脱敏回归通过；保活后 409 保持 HTTP 200 并发送固定会话错误，未提交响应头时仍返回 HTTP 409。
- 定制与计费：渠道 409 的结构化、恶意字段、空体及非 JSON 响应、状态映射和禁止重试；隐藏倍率优先级和显式 0；跨渠道重新预扣、余额不足禁止发送、累计退款、异步快照及用户日志隐私均通过现有和扩展回归。
- 数据库：真实 SQLite 3.50.4、MySQL 8.0.42、PostgreSQL 17.11。分别验证完整启动初始化，从 `b038440ae`（rc.45-x-x.3）创建的数据库升级，连续两次新版本启动迁移；每种引擎均使用独立主库与日志库，核查原有选项、用量、日志和审计数据保留、审计复合索引、缓存列读写、缓存插入及累加和统计查询。数据库矩阵未跳过 MySQL/PostgreSQL。
- 前端：Bun 1.4.4 冻结锁文件安装；Vitest 201 个文件、2455 项测试通过，另一个原有 `node:test` 用量文件单独用 Bun 执行，4 项通过；类型检查和生产构建通过。受影响版本与定价同步文件补跑 62 项通过。77 个变动前端文件 lint 无错误（官方统计代码保留两条 warning），格式检查通过；复用现有界面组件。
- 版本与工作流：数字修订排序、官方基线优先、跨新旧格式、重置到 v1、非法修订号拒绝、历史构建元数据、完整附件筛选与编码链接测试通过；发布脚本接受新旧格式并拒绝标签不匹配；三个发布工作流通过 actionlint，标签筛选矩阵确认新旧自维护标签均只匹配 amd64 发布。


## rc.51 源码合并（2026-11-16，未发布）

- 从自维护 `8c980a85b`（rc.48 v1）合并官方 `v1.0.0-rc.51`，只使用标签对应提交，不引入官方 main 的后续提交。工作分支 `codex/merge-newapi-rc51`。
- 保留隐藏倍率及显式 0、公开倍率与实际扣费分离、异步计费快照、普通用户模型隐私、兑换历史、每日 API Key 用量、自定义接口权限、CPR 可信身份/令牌名称透传、自维护更新源与 amd64 发布约定。官方未触及的 98 个二开文件中，仅更新版本元数据和本合并记录，其余 96 个文件保持原样。
- 合并冲突保留 CPR 配置变化时关闭旧 WebSocket 的处理，同时接入官方撤销旧渠道凭据逻辑；保留每日统计 handler，同时接入官方余额和历史清理任务。Docker PGO 跟随官方 MoeForge 模块名，保留 X-X 构建提交注入。
- 新插件路由保留命名空间补入二开 `/usage` 页面，防止插件接管用量页。新增提供商插件集成回归，验证隐藏 0.3、显式 0、公开倍率 1，以及宽重试规则和 409→500 状态映射下仍只请求一次、返回脱敏 `session_error/session_unavailable`。
- 新增官方测试的客户端原始错误断言适配为既有 X-X 固定错误契约；诊断细节改在内部错误日志断言，仍验证请求拒绝不发送上游、费用退款及各协议错误格式。未放宽客户端脱敏逻辑。
- 官方敏感词设置仍按 rc.51 迁移为请求过滤实例；普通模型接口继续使用既有 X-X 错误包装，不直接暴露官方新增的原始拒绝文案。此迁移、系统任务保留期、多节点版本兼容性仍是未来生产部署前的核查项，不能假定切回旧容器即可完整回滚。
- 未连接生产服务器、执行历史用量补算或更改注册/隐藏倍率排期；运维仓库已有修改保持原样。人工调整统计周期不得在未来升级验收中自动补算覆盖。
- 本次只做源码合并与本地验证，不推送发布标签、不创建 Release、不构建发布镜像、不部署。


### rc.51 本地验证与边界

- 工具链：Go 1.26.10、Bun 1.4.4；前端冻结锁文件安装。执行 Go 命令时设置 `GOWORK=off GOSUMDB=sum.golang.org`，避免本机 `GOSUMDB=off` 阻止匹配工具链校验；未修改全局工具配置。
- 根模块 `go test ./...`、`go build ./...`、`go vet ./...`；独立 `relaykit/`、`tokenkit/` 各自的相同三项验证通过。受影响 controller/router 在测试夹具修复后再次验证。
- `go test -race -p 2 ./relay/channel ./relay ./controller ./service -run 'CPR|Hidden|409|KeepAlive|Stream.*(Cancel|Error|Fail)' -count=1` 通过。提供商插件隐藏倍率/409 集成回归通过。
- 前端 `bun run test`：238 文件、2917 项通过；`bun run typecheck`：1506 文件、2 个项目通过；`bun run build` 通过。215 个受影响 TS/TSX/JS/JSX 文件 oxlint 无错误，手工解决冲突的 channel-form 格式检查通过。Bun 子命令的 PATH 指向隔离的 1.4.4，不使用系统 1.3.14。
- 数据库：应用/Go 驱动 SQLite 3.50.4、MySQL 8.0.46、PostgreSQL 17.12。设置标准 `TEST_MYSQL_DSN` / `TEST_POSTGRES_DSN` 运行 model/controller 的 `DatabaseMatrix|DatabaseRoundTrip|MigrationSchemaStability|PreviousRefreshHashMigrationConfigured`，三个引擎均实际执行；审计矩阵另设 `AUDIT_MYSQL_DSN` / `AUDIT_POSTGRES_DSN`，兑换码批量删除使用独立空库，均通过。`TestScheduledChannel*` 另外以 `TEST_TASK_DB_DIALECT=mysql` 和 `postgres` 验证行锁/取消回滚。
- 三种数据库各演练新建及从自维护 rc.48 v1 升级，共 6 组；每组新版重复启动两次，核对结构和数据不变。MySQL/PostgreSQL 分离主库和日志库；SQLite 采用应用现有共享文件模式。合成样本包含隐藏 0.3/显式免费、关闭注册、历史人工统计样本及含内部倍率的消费日志，不读取或导入生产数据。Python SQLite 查询器版本 3.40.1 只用于读写演练样本，不是应用 SQLite 引擎版本。
- 本地 `/api/status` 烟测核对嵌入版本 `v1.0.0-rc.51+lqy007700+v1`、官方基线 `v1.0.0-rc.51`。未发布版本元数据不代表已存在 Release。
- 曾在混合外部数据库的全量运行中出现要求空库的用例冲突，以及一次 WebSocket 额度耗尽时序断言失败；数据库用例改用独立库，并修复兑换测试缺少旧凭据截止时间初始化的前序依赖。WebSocket 用例独立重复 10 次、后续普通全量测试均通过，未为该偶发失败修改核心转发/鉴权逻辑；不据此声称消除了所有时序风险。
- 认证相关合并审阅采用 OWASP Authentication / Session Management Cheat Sheets；验证权限边界、凭据隔离和会话回归，未作完整 ASVS 合规认证。未验证 ClickHouse、最低支持版本 MySQL 5.7.8 / PostgreSQL 9.6，亦未执行真实支付或真实上游模型调用。
- 本次原始日志、迁移演练脚本与结果保存在 `/private/tmp/newapi-rc51-check/`；该目录为临时本地证据，不是生产备份，也不保证长期保留。未执行任何生产迁移或历史补算。
