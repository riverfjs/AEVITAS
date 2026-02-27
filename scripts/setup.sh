#!/usr/bin/env bash
set -euo pipefail

CONFIG_DIR="${HOME}/.aevitas"
CONFIG_FILE="${CONFIG_DIR}/config.json"

echo "=== aevitas setup ==="
echo ""

# Check if config exists
if [ -f "$CONFIG_FILE" ]; then
    echo "Config already exists: $CONFIG_FILE"
    read -rp "Overwrite? [y/N] " overwrite
    if [[ ! "$overwrite" =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 0
    fi
fi

# Provider
echo ""
echo "--- Provider ---"
read -rp "Provider type [anthropic/openai] (default: anthropic): " PROVIDER_TYPE
PROVIDER_TYPE="${PROVIDER_TYPE:-anthropic}"

read -rp "API Key: " API_KEY
read -rp "Base URL (leave empty for default): " BASE_URL

# Feishu
echo ""
echo "--- Feishu Channel ---"
read -rp "Enable Feishu? [y/N]: " FEISHU_ENABLED
if [[ "$FEISHU_ENABLED" =~ ^[Yy]$ ]]; then
    FEISHU_ENABLED="true"
    read -rp "App ID: " FEISHU_APP_ID
    read -rp "App Secret: " FEISHU_APP_SECRET
    read -rp "Verification Token (leave empty to skip): " FEISHU_VTOKEN
    read -rp "Webhook port (default: 9876): " FEISHU_PORT
    FEISHU_PORT="${FEISHU_PORT:-9876}"
else
    FEISHU_ENABLED="false"
    FEISHU_APP_ID=""
    FEISHU_APP_SECRET=""
    FEISHU_VTOKEN=""
    FEISHU_PORT="9876"
fi

# Telegram
echo ""
echo "--- Telegram Channel ---"
read -rp "Enable Telegram? [y/N]: " TG_ENABLED
if [[ "$TG_ENABLED" =~ ^[Yy]$ ]]; then
    TG_ENABLED="true"
    read -rp "Bot Token: " TG_TOKEN
else
    TG_ENABLED="false"
    TG_TOKEN=""
fi

# WeCom Bot
echo ""
echo "--- WeCom Bot Channel ---"
read -rp "Enable WeCom bot? [y/N]: " WECOM_ENABLED
if [[ "$WECOM_ENABLED" =~ ^[Yy]$ ]]; then
    WECOM_ENABLED="true"
    read -rp "Token: " WECOM_TOKEN
    read -rp "EncodingAESKey (43 chars): " WECOM_AES_KEY
    read -rp "ReceiveID (optional, leave empty to skip strict check): " WECOM_RECEIVE_ID
    read -rp "Callback port (default: 9886): " WECOM_PORT
    WECOM_PORT="${WECOM_PORT:-9886}"
else
    WECOM_ENABLED="false"
    WECOM_TOKEN=""
    WECOM_AES_KEY=""
    WECOM_RECEIVE_ID=""
    WECOM_PORT="9886"
fi

# Write config
mkdir -p "$CONFIG_DIR"

cat > "$CONFIG_FILE" <<EOF_JSON
{
  "agent": {
    "workspace": "${HOME}/.aevitas/workspace",
    "model": "claude-sonnet-4-5-20250929",
    "maxTokens": 8192,
    "temperature": 0.7,
    "maxToolIterations": 20
  },
  "provider": {
    "type": "${PROVIDER_TYPE}",
    "apiKey": "${API_KEY}",
    "baseUrl": "${BASE_URL}"
  },
  "channels": {
    "telegram": {
      "enabled": ${TG_ENABLED},
      "token": "${TG_TOKEN}",
      "allowFrom": [],
      "proxy": ""
    },
    "feishu": {
      "enabled": ${FEISHU_ENABLED},
      "appId": "${FEISHU_APP_ID}",
      "appSecret": "${FEISHU_APP_SECRET}",
      "verificationToken": "${FEISHU_VTOKEN}",
      "encryptKey": "",
      "port": ${FEISHU_PORT},
      "allowFrom": []
    },
    "wecom": {
      "enabled": ${WECOM_ENABLED},
      "token": "${WECOM_TOKEN}",
      "encodingAESKey": "${WECOM_AES_KEY}",
      "receiveId": "${WECOM_RECEIVE_ID}",
      "port": ${WECOM_PORT},
      "allowFrom": []
    }
  },
  "tools": {
    "braveApiKey": "",
    "execTimeout": 60,
    "restrictToWorkspace": true
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  }
}
EOF_JSON

chmod 600 "$CONFIG_FILE"

echo ""
echo "Config written to: $CONFIG_FILE"

# Initialize workspace
WORKSPACE_DIR="${HOME}/.aevitas/workspace"
echo ""
echo "Initializing workspace: $WORKSPACE_DIR"
mkdir -p "${WORKSPACE_DIR}/memory"
mkdir -p "${WORKSPACE_DIR}/.claude/skills"

# Copy template files if they don't exist
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

[ ! -f "${WORKSPACE_DIR}/AGENTS.md" ] && cp "${PROJECT_ROOT}/workspace/AGENTS.md" "${WORKSPACE_DIR}/" 2>/dev/null || true
[ ! -f "${WORKSPACE_DIR}/SOUL.md" ] && cp "${PROJECT_ROOT}/workspace/SOUL.md" "${WORKSPACE_DIR}/" 2>/dev/null || true
[ ! -f "${WORKSPACE_DIR}/.claude/settings.json" ] && cp "${PROJECT_ROOT}/workspace/.claude/settings.json" "${WORKSPACE_DIR}/.claude/" 2>/dev/null || true
[ ! -f "${WORKSPACE_DIR}/memory/MEMORY.md" ] && touch "${WORKSPACE_DIR}/memory/MEMORY.md"
[ ! -f "${WORKSPACE_DIR}/HEARTBEAT.md" ] && touch "${WORKSPACE_DIR}/HEARTBEAT.md"

# Copy skills if they don't exist
if [ -d "${PROJECT_ROOT}/workspace/.claude/skills" ]; then
    for skill in "${PROJECT_ROOT}/workspace/.claude/skills"/*; do
        if [ -d "$skill" ]; then
            skill_name=$(basename "$skill")
            if [ ! -d "${WORKSPACE_DIR}/.claude/skills/${skill_name}" ]; then
                cp -r "$skill" "${WORKSPACE_DIR}/.claude/skills/"
                echo "Installed skill: ${skill_name}"
            fi
        fi
    done
fi

echo "Workspace ready: $WORKSPACE_DIR"

echo ""
echo "Next steps:"
echo "  make gateway    # Start gateway"
if [ "$FEISHU_ENABLED" = "true" ]; then
    echo "  make tunnel     # Start cloudflared tunnel for Feishu webhook"
fi
if [ "$WECOM_ENABLED" = "true" ]; then
    echo "  Configure callback URL to /wecom/bot"
fi
echo ""
echo "Done."
