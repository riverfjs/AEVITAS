'use strict';
/**
 * notify.cjs — Real-time progress notifications via myclaw gateway WebSocket RPC.
 *
 * Sends a notify.send RPC call over WebSocket so the gateway can forward
 * the message to Telegram (or whichever channel is configured) while the
 * agent is still waiting for the skill's Bash tool to finish.
 *
 * Requires:
 *   - Node.js 21+ (uses the built-in global WebSocket — no npm packages needed)
 *   - myclaw gateway running on the port configured in ~/.myclaw/config.json
 *   - notify.send registered on the gateway (internal/rpc/notify.go)
 *
 * Wire protocol:
 *   { type:"req", id, method:"notify.send", params:{ channel, chatId, message } }
 *
 * Usage:
 *   const { notify } = require('./notify.cjs');
 *   await notify('⏳ Fetching financials…');
 */

const fs   = require('fs');
const os   = require('os');
const path = require('path');

/** Read gateway URL and Telegram chatId from ~/.myclaw/config.json. */
function loadConfig() {
  try {
    const raw    = JSON.parse(fs.readFileSync(path.join(os.homedir(), '.myclaw', 'config.json'), 'utf8'));
    const host   = raw?.gateway?.host ?? '127.0.0.1';
    const port   = raw?.gateway?.port ?? 18790;
    // Bind address 0.0.0.0 means "all interfaces" — connect via loopback.
    const wsHost = (host === '0.0.0.0' || host === '') ? '127.0.0.1' : host;
    // Use the first allowed Telegram user as the notification target.
    const chatId = String(raw?.channels?.telegram?.allowFrom?.[0] ?? '');
    return { wsUrl: `ws://${wsHost}:${port}`, chatId, channel: chatId ? 'telegram' : '' };
  } catch {
    return { wsUrl: 'ws://127.0.0.1:18790', chatId: '', channel: '' };
  }
}

/**
 * Push a progress message to Telegram via the gateway notify.send RPC.
 *
 * Silently no-ops if:
 *   - chatId is not configured in config.json
 *   - gateway is unreachable (timeout 3s)
 *   - any WebSocket error occurs
 *
 * @param {string} message  Text to send (plain text, no Markdown needed)
 */
async function notify(message) {
  const { wsUrl, chatId, channel } = loadConfig();
  if (!chatId) return; // not configured — skip

  await new Promise(resolve => {
    let done  = false;
    let opened = false;
    const finish = () => { if (!done) { done = true; resolve(); } };
    let ws;
    // Give the round-trip at most 1.2 seconds; skip silently on timeout.
    const timer = setTimeout(() => {
      try {
        if (ws) ws.close(1000, 'timeout');
      } catch {}
      finish();
    }, 1200);

    try {
      ws = new WebSocket(wsUrl); // Node.js 21+ global — no import needed
    } catch {
      clearTimeout(timer);
      finish();
      return;
    }

    ws.addEventListener('open', () => {
      opened = true;
      ws.send(JSON.stringify({
        type:   'req',
        id:     `notify-${Date.now()}`,
        method: 'notify.send',
        params: { channel, chatId, message },
      }));
      // Give gateway a brief moment to receive before closing client side.
      setTimeout(() => {
        try { ws.close(1000, 'done'); } catch {}
      }, 80);
    });

    // Close as soon as we receive the response — we don't need to read it.
    ws.addEventListener('message', () => {
      clearTimeout(timer);
      try { ws.close(1000, 'done'); } catch {}
      finish();
    });
    ws.addEventListener('error',   () => { clearTimeout(timer); finish(); });
    ws.addEventListener('close',   () => {
      clearTimeout(timer);
      // If close happens before open, treat as transient and ignore.
      if (!opened) return finish();
      finish();
    });
  });
}

module.exports = { notify };
