# 飞书 Bot 配置教程（长连接模式）

## 前置条件

- 飞书账号（企业自建应用）
- aevitas 已编译（`make build`）
- 运行 aevitas 的机器可访问公网（用于建立飞书 SDK WebSocket 长连接）

> 长连接模式不需要自建域名、不需要 webhook 回调地址、也不需要内网穿透。

## 第一步：创建飞书应用

1. 登录 [飞书开放平台](https://open.feishu.cn/)
2. 创建「企业自建应用」
3. 进入「凭证与基础信息」，记录：
   - `App ID`
   - `App Secret`

## 第二步：添加机器人能力与权限

1. 在「应用能力」中添加 **机器人**
2. 在「权限管理」中开通：

| 权限 | 说明 |
|------|------|
| `im:message` | 获取与发送消息 |
| `im:message:send_as_bot` | 以机器人身份发送消息 |

3. 发布应用版本（权限变更后必须重新发布）

## 第三步：配置事件订阅

1. 进入「事件与回调」->「事件配置」
2. 添加事件：`im.message.receive_v1`

> 长连接模式下无需填写请求地址（Webhook URL）。

## 第四步：配置 aevitas

编辑 `~/.aevitas/config.json`：

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "appId": "cli_xxxxx",
      "appSecret": "your-app-secret",
      "allowFrom": []
    }
  }
}
```

可选环境变量覆盖：

```bash
export AEVITAS_FEISHU_APP_ID="cli_xxxxx"
export AEVITAS_FEISHU_APP_SECRET="your-app-secret"
```

## 参数说明

| 参数 | 类型 | 说明 |
|------|------|------|
| `enabled` | bool | 是否启用飞书通道 |
| `appId` | string | 飞书应用 App ID |
| `appSecret` | string | 飞书应用 App Secret |
| `allowFrom` | []string | 允许的 open_id 列表（空=允许所有人） |

## 第五步：启动并验证

```bash
make gateway
```

启动日志看到以下内容即表示长连接建立成功：

```text
[gateway] channels configured: [telegram feishu]
[channel-mgr] starting feishu
[feishu] long connection started
```

## 常见问题

**Q: 飞书消息收不到？**

- 确认应用已发布最新版
- 确认已添加 `im.message.receive_v1` 事件
- 确认运行机器可访问公网
- 确认机器人已被添加到目标会话/群聊

**Q: Access denied？**

- 检查权限是否已开通并发布
- 检查 `allowFrom` 是否误限制了 open_id

**Q: 如何获取 open_id？**

- 查看 aevitas 日志中的飞书入站信息，`ou_xxx` 即 open_id
