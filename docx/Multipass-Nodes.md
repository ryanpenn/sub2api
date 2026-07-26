# Multipass Ubuntu 节点信息

核验日期：2026-07-26（Asia/Shanghai）

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
