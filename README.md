# 新蜂资产管理平台 — 统一 I/O RPC 服务

负责数据源 Provider、采集与转换、任务队列、Worker 和定时任务等处理逻辑，通过 Core、CMDB 和 Ops RPC 与平台其他服务协作。此模块属于统一 I/O，不是 CMDB RPC。

仓库：[coder-lulu/newbee-io-rpc](https://github.com/coder-lulu/newbee-io-rpc) · [平台工作区](https://github.com/coder-lulu/newbee)

## 获取代码

推荐通过完整工作区开发，保留兄弟模块目录及本地 `replace` 依赖。以下命令使用 Bash；Go 工作区要求 Go 1.25.1 或更高版本。

```bash
git clone --recurse-submodules https://github.com/coder-lulu/newbee.git
cd newbee/unified-io/rpc
```

已有工作区执行 `git submodule update --init --recursive`。单独克隆模块时，需要自行补齐 `go.mod` 中的本地依赖路径。

## 目录导航

| 路径 | 用途 |
| --- | --- |
| `io.go` | 服务入口 |
| `ent/` | 数据模型及生成代码 |
| `internal/logic/`、`internal/svc/` | 业务逻辑与依赖装配 |
| `internal/server/`、`types/` | RPC 实现和协议类型 |
| `etc/io.yaml.example` | 公开配置模板 |

## 配置与本地运行

首次配置时执行下列复制命令；已有配置不要覆盖：

```bash
cp etc/io.yaml.example etc/io.yaml
```

按环境修改 `DatabaseConf、RedisConf、CoreRpc、CmdbRpc、OpsRpc、TaskWorker 和 InputAdapterConf` 等配置项，以及监听地址。示例 RPC 端口为 `9500`，以实际配置为准。入口使用 `conf.UseEnv()`，可为示例中的 `${...}` 占位符设置对应环境变量，也可在本地配置中填写值。模板中的内网地址不是可直接使用的公共服务。

先准备数据库、Redis 和配置引用的 RPC 服务，再启动本服务；API 应在对应 RPC 就绪后启动。真实配置和凭据不要提交。

```bash
go run . -f etc/io.yaml
```

## 首次数据库初始化

先初始化 Core，并配置 `CoreRpc`（支持 `Endpoints`、`Target` 或 Etcd 服务发现）。在平台根目录运行 `bash init-databases.sh -s unified-io`。首次初始化创建 Ent 表及发现 Provider 基础配置，再通过 Core RPC 登记当前 API 目录和 I/O 菜单；菜单按路径或组件复用，不依赖固定父菜单 ID。默认 `superadmin` 的菜单授权合并保留，普通角色需单独授权。

Core 不可用、默认角色缺失或目录登记失败会返回错误；修复后可重复初始化。初始化阶段保持 `TaskWorker.Enabled` 关闭（同时关闭任务 Worker 与调度器），在初始化完成并配置好采集目标后按需启用。请在启动 API 前完成初始化，无需导入本机数据库备份或历史菜单 SQL。

## 构建与验证

在当前模块目录执行：

```bash
go build .
go test ./...
go vet ./...
```

测试中的集成用例需要对应基础服务。修改协议或 Ent Schema 后，应使用当前 Makefile 中对应生成目标并审查生成差异；不要直接编辑生成文件。上述命令是验证入口，不表示所有测试已通过。

## 文档

[Provider 插件指南](docs/PROVIDER_PLUGIN_IMPLEMENTATION_GUIDE.md) · [定时任务指南](docs/CRON_USER_GUIDE.md) · [Worker 说明](internal/worker/README.md) · [集成测试说明](internal/integration_test/README.md)

Provider 公共接口位于 [newbee-io](https://github.com/coder-lulu/newbee-io)，采集结果与资产入库职责由统一 I/O 和 CMDB 协作承担。

## 许可证与来源

本仓库采用 [Apache-2.0](LICENSE)。沿用现有服务框架的上游许可，保留文件中的原作者版权。第三方依赖遵循各自许可证，保留原有版权与许可声明。
