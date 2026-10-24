# 小步数学

五年级数学的小范围诊断与练习原型。Go + Vue 3 + SQLite，六模块、12 个起步能力点、72 道原创题。不是完整课纲或标准化测评。

## 本地启动

需要 Go 1.24+、C 编译器（SQLite 驱动使用 CGO）和 Node.js 20.19+ / 22.12+。macOS 的 Xcode Command Line Tools 提供 C 编译器。

在仓库根目录执行：

```bash
bash apps/grade5-math/run.sh
```

首次启动安装前端和 Go 依赖，然后构建。打开终端打印的地址，默认是 http://127.0.0.1:8080。`Ctrl-C` 停止。设置 `MATH_ADDR=127.0.0.1:8081` 可换端口。

数据库保存在 `apps/grade5-math/var/math.db`，已被 Git 忽略。匿名 Cookie 关联当前浏览器，不保存真实姓名；重启服务保留进度，换浏览器不能自动同步。停止服务后备份整个 `var/` 目录可保留数据。默认仅监听本机。服务器通过现有 Nginx 反向代理访问。

## 前后端分别开发

终端一，在 `apps/grade5-math/backend`：

```bash
GOCACHE=/tmp/grade5-math-go-cache go run ./cmd/server
```

终端二，在 `apps/grade5-math/frontend`：

```bash
npm ci
npm run dev
```

打开 Vite 打印的地址。API 自动代理至 Go 的 8080 端口；开发和正式预览端口的 Cookie 使用各自访问时的浏览器主机，建议统一使用 `127.0.0.1`。

## 内容与诊断

- 一轮先从各已选模块取一道筛查题，再追加最多 4 道确认题，总计最多 10 题。新的体检会轮换模块内的能力点。
- 一道错题只触发“需要再确认”，两道独立题均错才建议巩固；“还没学过”和“暂时不会”不计为错误。
- 每次练习有 2 道带解释的题，再做 2 道新的复测题。练习不混入独立检测证据；两道复测均正确只表示“本次复测通过”。
- 同一题的重复请求不增加记录；同一会话不会重复使用题目。每个能力点在一轮中只练习一次，用完后可开始新体检。
- 服务器保存内容快照和数值精确判分结果；题库更新不会改变进行中的会话。会话和答题记录以 JSON 快照存入 SQLite，写入使用事务。
- 当前题目响应不包含答案和解析；完整检测结束后才开放回顾。练习可立即查看解释。
- 数字输入接受有限十进制；分数接受 `a/b`。一般分数按有理数等值判分，明确要求特定分母的题额外检查分母。
- 知识路线只表示原图关联，前置节点未单独检测时明确标注，不推断它是错误原因。

## 检查与重建

仓库根目录：

```bash
node apps/grade5-math/scripts/build-content.mjs --check
npm test
npm run validate
```

后端目录：

```bash
GOCACHE=/tmp/grade5-math-go-cache go test -race ./...
GOCACHE=/tmp/grade5-math-go-cache go run ./cmd/server -check
```

前端目录：

```bash
npm run build
```

内容源为 `scripts/build-content.mjs`：运行不带 `--check` 的命令会重建三份 `content/*.json`。脚本检查原图无环性，输出原始 topics/dependencies 文件哈希；`--check` 检查提交的内容是否可复现。不要直接编辑生成的 JSON。更新题目时同步变更题库 `version`，并更新后端的独立答案核对表。

服务器启动时检查引用、题型、库存、答案格式、选项及图循环。测试另含 72 道题的独立计算答案表；这不代替教师对表述、难度、教学适切性的审核。

## 范围与来源

候选内容横跨五年级常见数学模块，尚未正式对齐中国某版教材、学期或课程标准。当前题目和诊断规则需要教师审阅及实际学生使用验证，不输出综合能力分、排名或长期掌握率。

知识关系来自 Marble Skill Taxonomy v1，© Generative Spark, Inc. (Marble)，https://withmarble.com。数据库采用 ODbL 1.0，原作者文本采用 CC BY-SA 4.0，第三方标准遵循各自条款。应用 UI 有来源入口；具体信息见仓库根目录 README、LICENSE、LICENSE-CONTENT、PROVENANCE.md。本应用没有重新分发第三方课程标准全文。

详细设计：[../../docs/grade5-math-mvp.md](../../docs/grade5-math-mvp.md)。

## 服务器发布与上游更新

这是验证简单功能的试验项目，后续产品需求尚未确定。当前知识点、题型与诊断规则均是首版样例。

- [发布、回滚与 HTTPS 后续步骤](deploy/README.md)
- [同服务器站点台账与统一配置约定](deploy/SERVER.md)
- [上游同步流程](UPSTREAM.md)

应用、发布脚本和运行数据与原始知识图谱分离；不要把 SQLite、SSH 私钥或构建产物提交进 Git。
