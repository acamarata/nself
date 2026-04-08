# Feature: Agent Dashboard

The Agent Dashboard gives you a real-time view of all 8 nClaw personas and their activity. It shows live metrics, sparkline trends, and per-agent status so you can see what your AI assistant is doing at a glance.

**Route:** `/agents` in claw-web
**API:** `GET /claw/agents/dashboard` and `GET /claw/agents/status`

---

## What It Shows

Each persona card displays:

- Current status (active, idle, error)
- Messages processed in the last 24 hours
- Average response time
- Sparkline trend chart (7-day activity)
- Last active timestamp

The dashboard updates in real time via WebSocket. No manual refresh needed.

## The 8 Personas

| Persona | Focus Area |
|---------|-----------|
| CamClaw | General assistant, email, calendar, tools |
| Health Coach | Check-ins, habit tracking, health summaries |
| Learning Tutor | Flashcards, quizzes, spaced repetition |
| Daily Mentor | Reflections, goal nudges, journaling |
| Domain Expert | Configurable expertise (coding, finance, law, etc.) |
| Job Hunter | CV management, career suggestions, job search |
| nChat Persona | Chat-specific behaviors |
| Custom | User-defined via persona marketplace |

## Agent Crews

You can combine personas into crews that work together on multi-step tasks. A crew config defines which personas participate, their roles, and the workflow order.

**API:**
- `GET /claw/crews` -- list crew configs
- `POST /claw/crews` -- create a new crew

**Route:** Crew management is available from the Agent Dashboard page.

## Configuration

No special configuration needed. The dashboard reads from the existing persona system. All installed personas appear automatically.

## Related

- [[Feature-nClaw]] -- nClaw overview
- [[Feature-Memory-Rooms]] -- Memory organization
- [[Feature-Image-Generation]] -- Image generation
