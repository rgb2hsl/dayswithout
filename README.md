# dayswithout

Telegram bot that counts how many days have passed since the last mention of a specific topic in a group chat.  
Supports manual reset via command and soft triggers on custom keywords.

---

## ✨ Features

- Group chat support.
- Configurable **topic** and **keywords** in `config.yaml`.
- Commands:
  - `/days` — show how many days have passed since the last mention and when it was.
  - `/reset` — reset the counter (record current time as last mention).
- Soft keyword detection:
  - If a keyword is mentioned in the chat, the bot **asks if the counter should be reset**, but does not reset automatically.
- "Cooldown": bot ignores repeated triggers for 2 hours after the last mention.
- Simple file-based storage (`data.json`).
- Deployable as a **systemd service** on Ubuntu.

---

## ⚙️ Configuration

Create a `config.yaml` file in the project root:

```yaml
bot_token: "%YOUR_TG_BOT_TOKEN%"

# Topic
topic: "Fruits"

# Triggers
keywords:
  - "(?i)apple\\w*"
  - "(?i)orange\\w*"
  - "(?i)banana\\w*"
  - "(?i)fruit\\w*"
  - "(?i)вэриен\\w*"
