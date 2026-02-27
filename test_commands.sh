#!/bin/bash
# 测试 aevitas 命令处理功能

echo "=== aevitas 命令功能测试 ==="
echo ""
echo "已添加的命令："
echo "  /start  - 显示欢迎消息"
echo "  /help   - 显示帮助信息"
echo "  /reset  - 重置会话历史"
echo ""
echo "这些命令在 Telegram/Feishu/WeCom 等 channel 中可用"
echo "通过 gateway 模式运行时自动激活"
echo ""
echo "启动 gateway："
echo "  cd /Users/fanjinsong/Documents/chatbot/aevitas && make gateway"
echo ""
echo "然后在 Telegram 中测试："
echo "  /start  - 应显示 Aevitas 欢迎信息"
echo "  /help   - 应显示命令列表"
echo "  /reset  - 应清除对话历史"

