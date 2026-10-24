# 同机站点台账与约定

只读核查日期：2026-10-24（Asia/Shanghai）。服务器为 OpenCloudOS 8.10 / x86_64，约 2 GB 内存，已有 Nginx 1.14.1、systemd、Certbot、SQLite CLI。以下路径来自实际 Nginx 配置和 service 状态，不仅依据 server_name 搜索。

| 站点 | 实际入口 | 前端/页面 | 程序、数据及服务 |
| --- | --- | --- | --- |
| www.idaay.com | HTTP 默认站点，标题“千年等一壶” | `/usr/share/nginx/html/index.html` | 静态站，无独立 Go 服务 |
| mkd-app.idaay.com | HTTPS；HTTP 301 跳转 | `/var/www/mkd-app-current` → `/var/www/mkd-app-20260918-demo1` | `/opt/mkd-app/mkd-app-server`；`/var/lib/mkd-app`；`mkd-app.service`；用户 `mkdapp`；`127.0.0.1:18080`；内存上限 256 MiB |
| start.deey.top | 本项目独立 HTTPS 虚拟主机；HTTP 跳转 | `/var/www/grade5-math-current` | `/opt/grade5-math/current`；`/var/lib/grade5-math`；`grade5-math.service`；用户 `grade5math`；`127.0.0.1:18081`；内存上限 192 MiB |

旧站配置：

- www 静态站写在 `/etc/nginx/nginx.conf` 的默认 server 中，`server_name _`；因此单看命名虚拟主机会漏掉它。
- mkd 使用 `/etc/nginx/conf.d/mkd-app.idaay.com.conf`，证书 SAN 只有 `mkd-app.idaay.com`。
- 部署前访问 start 的 HTTP 也落在默认静态站，页面与 www 相同。这并不说明 start 已部署应用。
- www 的 HTTPS 在部署前即证书名称不匹配，不把这个历史问题归因于本次部署，也不擅自修复。
- 远程 DNS 确认 www、mkd-app、start 指向 `203.195.200.128`；本机代理 DNS 返回 `198.18.*` 假 IP，不能据此判断公网解析。idaay.com 裸域未在远程系统正常解析，未作为受保护的已运行站点认定。

## 统一约定

现有动态站和本项目遵循 `/opt/<项目>` 放程序、`/var/www/<项目>-current` 放前端入口、`/var/lib/<项目>` 放可写数据、独立用户/服务/本机端口、`/etc/nginx/conf.d/<域名>.conf` 放精确虚拟主机的同一模型。新程序增加版本目录和原子软链接，以便回滚。

默认静态首页是现有遗留布局，明确登记并保持原状。若未来统一它的目录和证书，作为单独迁移任务做；本次不移动旧页面、不修改 nginx.conf，不移动 mkd 程序/数据库，不更新系统包或旧服务。

每次上线记录前后配置哈希、两旧站首页内容哈希、新应用 revision、服务资源和回滚位置。原始 Nginx 备份可能含私密信息，只存服务器 root 可读备份目录，不提交进公共仓库。以后其他 AI 操作此机，先读此台账并重新核查实时配置；不把这份历史快照当作永远有效。

## HTTPS 状态补充（2026-10-24）

start.deey.top 已配置独立 Let’s Encrypt 证书，复用 `certbot-renew.timer`；模拟续期成功，专属重载钩子验证通过。mkd 与默认站点的配置、mkd 证书和续期文件哈希保持不变。www HTTPS 的历史证书名称不匹配仍未在此任务中处理。
