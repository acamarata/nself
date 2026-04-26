# Moderation Plugin

> Content moderation, profanity, toxicity, AI review, rule policies, appeals, and reports. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install moderation
```

## What It Does

Provides a multi-layer content moderation system for user-generated content. Filters profanity with custom word lists, detects toxicity via ML classifiers, routes flagged content for human review, defines rule-based auto-moderation policies, handles user appeals, and manages user reports. Supports pre-publish and post-publish moderation modes.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MODERATION_PORT` | `3208` | Moderation service port |
| `MODERATION_MODE` | `post` | Moderation timing: `pre` or `post` publish |
| `MODERATION_AI_ENABLED` | `false` | Use AI plugin for toxicity scoring |
| `MODERATION_AUTO_REMOVE_THRESHOLD` | `0.95` | Auto-remove above this toxicity score |
| `MODERATION_PROFANITY_ACTION` | `flag` | Action: `flag`, `replace`, or `remove` |

## Ports

| Port | Purpose |
|------|---------|
| 3208 | Moderation REST API |

## Database Tables

18 tables added to your Postgres database:
- `np_moderation_items`, content pending review
- `np_moderation_decisions`, moderation decisions
- `np_moderation_policies`, auto-moderation rules
- `np_moderation_word_lists`, profanity word lists
- `np_moderation_appeals`, user appeals
- `np_moderation_reports`, user-submitted reports
- `np_moderation_reviewers`, moderator assignments
- `np_moderation_queue`, review queue state
- And 10 more for audit, stats, actions, exemptions, etc.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/moderation/` | Moderation management API |
