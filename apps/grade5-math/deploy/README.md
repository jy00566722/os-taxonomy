# 小步数学：本地编译、服务器运行

这是简单功能验证项目。当前地址为 https://start.deey.top；HTTP 自动跳转，ACME 验证目录保留 HTTP 访问。HTTPS 已于 2026-10-24 启用。

## 发布模型

- **Mac 本地**：构建 Vue；Docker Buildx 使用固定官方 Go 1.24.3 镜像，为 `linux/amd64` 编译静态 CGO/SQLite 程序，并执行一次后端核心测试。
- **服务器**：只上传压缩包，不安装 Docker、Go、Node 或编译器；使用已有 Nginx + 独立 systemd 服务。
- `Dockerfile.build` 只是一份本地跨架构编译配方，**不是服务器容器配置**。
- 每个包记录 Git revision、构建时间、平台和文件 SHA-256。运行态 `/api/health` 返回 revision，可核对实际发布版本。
- 发布必须来自已提交的干净工作区；本机 `.cache/`、`var/`、`deploy/.artifacts/` 和前端产物均不进入 Git。

## 目录和资源

| 用途 | 服务器路径/值 |
| --- | --- |
| 程序、内容和按版本归档的前端 | `/opt/grade5-math/releases/<release>/` |
| 当前版本 | `/opt/grade5-math/current` 原子切换软链接 |
| Nginx 前端根目录 | `/var/www/grade5-math-current` → `/opt/grade5-math/current/web` |
| SQLite 持久化数据 | `/var/lib/grade5-math/math.db` |
| 服务 | `/etc/systemd/system/grade5-math.service`，用户 `grade5math` |
| 后端 | `127.0.0.1:18081`，仅本机监听 |
| 站点配置 | `/etc/nginx/conf.d/start.deey.top.conf` |
| 备份和验收基线 | `/var/backups/grade5-math/<release>/`，仅 root 可读 |
| 日志 | journald；Nginx `grade5-math.access.log` / `grade5-math.error.log` |
| 资源 | `GOMAXPROCS=1`、Go 软内存目标 96 MiB、systemd 内存上限 192 MiB、CPU 配额 75% |

内存目标/上限不是预估实际用量，部署后以 `systemctl show` 实测。程序从当前版本只读加载内容，数据库在版本目录之外，更新不覆盖数据。

## 一次发布

在仓库根目录提交应用、文档和发布脚本。当前开发分支为 `codexY/grade5-math-check`，推送到自己的 `origin`；不向上游推送。

```bash
git add apps/grade5-math docs/grade5-math-mvp.md
git commit -m "feat: add grade5 math prototype and isolated server deployment"
git push -u origin codexY/grade5-math-check
bash apps/grade5-math/deploy/build.sh
```

build 脚本最后输出 `deploy/.artifacts/<UTC时间>-<Git短哈希>.tgz`。把该绝对路径传入：

```bash
bash apps/grade5-math/deploy/upload.sh /absolute/path/to/release.tgz
```

默认连接 `root@start.deey.top:22`，私钥 `~/.ssh/id_rsa`；可通过 `DEPLOY_TARGET`、`DEPLOY_PORT`、`DEPLOY_KEY` 覆盖。私钥留在本机，不上传、不写入仓库。

远程 activate 脚本加发布锁，先保存 Nginx 配置和两旧站内容指纹；已有数据库用 SQLite `.backup` 备份。校验包与内容，创建新版本目录，再切换本应用软链接和服务。只有初次建站时安装本应用 Nginx 配置，并在 `nginx -t` 成功后平滑 reload。常规应用发布不覆盖已上线的 Nginx 文件，避免未来把 HTTPS 配置改回 HTTP。

发布后仅检查本应用健康、首页和两个旧站的原状态。旧站配置和默认首页文件 SHA-256 必须不变；只重启 `grade5-math`，Nginx 只平滑 reload。

## 回滚

同一发布包可重试：已解包目录必须与包内 SHA256SUMS 一致，再验证全部文件，不覆盖已有版本文件；每次尝试另存备份。Nginx reload 为异步平滑切换，脚本会等待该域名健康接口返回预期 revision 后再判定完成。

发布脚本失败时尝试恢复本应用旧软链接、旧服务文件和服务；初次新增的 Nginx 文件会移入本次备份，再校验和 reload。失败版本和备份保留排查。

手动回滚到一个已知版本时，先核对 `current`、目标版本和目标 `RELEASE.json`。在服务器执行：

```bash
# 把占位路径替换为实际存在、已核对的旧版本。
ln -s /opt/grade5-math/releases/OLD_RELEASE /opt/grade5-math/current.rollback
mv -Tf /opt/grade5-math/current.rollback /opt/grade5-math/current
systemctl restart grade5-math
curl -fsS http://127.0.0.1:18081/api/health
```

不自动恢复数据库：当前会话内容自带快照，单纯程序回滚不应丢弃新作答。未来涉及数据库结构变化时另做迁移/恢复方案。回滚应用版本不需要 reload Nginx。

## HTTPS 与自动续期（已启用）

- 独立 Let’s Encrypt 免费证书：`/etc/letsencrypt/live/start.deey.top/`，本轮到期日 2027-01-22。
- Nginx HTTPS 模板：`start.deey.top.https.conf`，只覆盖本域名文件，复用 mkd 现有 TLS 参数和 DH 参数文件。
- 在 `/etc/sysconfig/grade5-math` 设置 `MATH_EXTRA_FLAGS=-secure-cookies`，新发匿名 Cookie 带 Secure。
- 认证方式：webroot，目录 `/var/www/grade5-math-acme`；独立续期配置 `/etc/letsencrypt/renewal/start.deey.top.conf`。
- 复用服务器已有并启用的 `certbot-renew.timer`，不创建第二套 cron/timer。它每天两个计划时段检查，含随机延迟，仅需要续签时才更新证书。
- 新证书的专属 `renew_hook` 指向 `/opt/grade5-math/ops/certbot-deploy.sh`：仅处理本域名，先 `nginx -t`，成功再平滑 reload。这个目录不随应用版本回滚消失。
- 2026-10-24 新证书模拟续期通过；安装的 Certbot 1.22.0 在 dry-run 时不执行 deploy hook，本次另外执行一次钩子，校验和 reload 成功。
- 后续应用发布脚本识别当前站点 HTTP/HTTPS，再选择相应本机入口检查，不把 HTTP 301 误认为发布失败。常规发布仍不覆盖本域名 Nginx 配置。

初次配置脚本为 `enable-https.sh`，配套模板和钩子先上传到该脚本指定的 `/var/tmp/grade5-math-*` 路径，再通过 SSH 执行。已启用的服务器不需要重复执行。

mkd 原方案是 Let’s Encrypt + Certbot nginx 插件 + 同一个 systemd timer，属于常规方案；共享定时任务最近执行成功。新站复用 CA、timer 和 TLS 参数，改用 webroot 和专属重载钩子，以便手工管理的虚拟主机不被插件自动重写。没有修改 mkd 的配置、证书或续期参数。旧站原文件哈希均保持一致。

HTTPS 配置备份：`/var/backups/grade5-math/https-20261024T034629Z`。若回滚，恢复该目录的站点和环境配置，再重启本应用、校验并 reload Nginx；保留已申请证书，不删除共享续期任务。
