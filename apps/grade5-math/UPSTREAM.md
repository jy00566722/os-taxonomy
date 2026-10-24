# 原项目与上游更新

- 自有仓库 origin：`https://github.com/jy00566722/os-taxonomy.git`
- 原始项目 upstream：`https://github.com/withmarbleapp/os-taxonomy.git`
- 应用分支：`codexY/grade5-math-check`
- 首版应用使用的知识图谱基线提交：`95e6ddd`。上游已出现更新，本次部署不混入未经分析的数据更新。

## 目录边界

原始 `data/`、`schema/`、`scripts/`、根 package.json 和 README 保持上游版本；本应用主要在 `apps/grade5-math/`，原型设计在 `docs/grade5-math-mvp.md`。不复制整个上游到另一份仓库，不删除其许可/来源文件。构建包只携带应用所需的小图谱投影，不上传原始约 24 MB 教学研究全量文件。

本地数据库、缓存、Linux 二进制、前端构建和服务器原始备份不跟踪；SSH 私钥永不入库。此分支只推到 origin。upstream 的 push URL 被设为禁用，防止误推。

## 一次上游更新

```bash
git fetch upstream main --no-tags
git log --oneline HEAD..upstream/main
git diff HEAD...upstream/main -- data schema scripts README.md
# 从已提交的应用分支另开同步分支，日期按实际填写。
git switch -c codexY/upstream-sync-YYYYMMDD
git merge upstream/main
```

先分析上游删除/合并了哪些 topic ID、依赖边和内容结构，再处理冲突。不要用强制覆盖、自动选一边或忽略 data 的办法消除冲突。若上游增加 AGENTS.md，先阅读其适用要求。

随后重建 `node apps/grade5-math/scripts/build-content.mjs`，核对生成差异；映射失效时调整中文能力点对应的 topic ID。这个脚本记录 topics/dependencies 哈希并检查循环；JSON 由脚本生成，不能手改后绕过校验。

核心验证是图谱投影检查、后端核心测试与前端构建。只有上游变更原始数据/校验器时才补跑原项目验证；不无条件扩展测试范围。确认后提交、推送同步分支，再通过正常的本地构建和上传流程发布。服务器不执行 git pull，也不自动追踪上游；合并与部署始终是两步。

同一服务持久化的每个会话包含内容快照，进行中的会话不会因新知识图谱投影而改判。新会话使用新发布版本。内容题库有变化时更新题库 version 和独立答案表。
