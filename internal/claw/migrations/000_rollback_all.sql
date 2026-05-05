-- S27-05: Consolidated rollback for claw plugin.
-- Drops all np_claw_* tables + _schema_version entries.
-- NOT REVERSIBLE: data loss — use only during controlled teardown.

BEGIN;

-- Drop knowledge graph + memory tables first (they reference sessions/users)
DROP TABLE IF EXISTS np_claw_kg_edges           CASCADE;
DROP TABLE IF EXISTS np_claw_kg_entities        CASCADE;
DROP TABLE IF EXISTS np_claw_memory_conflicts   CASCADE;
DROP TABLE IF EXISTS np_claw_memory_temporal    CASCADE;
DROP TABLE IF EXISTS np_claw_memory_history     CASCADE;
DROP TABLE IF EXISTS np_claw_memory_rooms       CASCADE;
DROP TABLE IF EXISTS np_claw_memories           CASCADE;
DROP TABLE IF EXISTS np_claw_prune_candidates   CASCADE;
DROP TABLE IF EXISTS np_claw_retrieval_profiles CASCADE;
DROP TABLE IF EXISTS np_claw_saved_searches     CASCADE;

-- Conversations
DROP TABLE IF EXISTS np_claw_message_summaries  CASCADE;
DROP TABLE IF EXISTS np_claw_messages           CASCADE;
DROP TABLE IF EXISTS np_claw_threads            CASCADE;
DROP TABLE IF EXISTS np_claw_sessions           CASCADE;
DROP TABLE IF EXISTS np_claw_branches           CASCADE;
DROP TABLE IF EXISTS np_claw_topics             CASCADE;
DROP TABLE IF EXISTS np_claw_touchpoints        CASCADE;
DROP TABLE IF EXISTS np_claw_briefings          CASCADE;

-- Agents / tasks
DROP TABLE IF EXISTS np_claw_agent_tasks        CASCADE;
DROP TABLE IF EXISTS np_claw_agent_queue        CASCADE;
DROP TABLE IF EXISTS np_claw_agents             CASCADE;
DROP TABLE IF EXISTS np_claw_teams              CASCADE;
DROP TABLE IF EXISTS np_claw_skills             CASCADE;
DROP TABLE IF EXISTS np_claw_playbooks          CASCADE;

-- Audit / security
DROP TABLE IF EXISTS np_claw_audit_log          CASCADE;
DROP TABLE IF EXISTS np_claw_approval_rules     CASCADE;
DROP TABLE IF EXISTS np_claw_pending_approvals  CASCADE;
DROP TABLE IF EXISTS np_claw_integration_probes CASCADE;

-- Users / identity
DROP TABLE IF EXISTS np_claw_user_vocabulary    CASCADE;
DROP TABLE IF EXISTS np_claw_identity_sessions  CASCADE;
DROP TABLE IF EXISTS np_claw_users              CASCADE;
DROP TABLE IF EXISTS np_claw_setup_tokens       CASCADE;
DROP TABLE IF EXISTS np_claw_api_keys           CASCADE;

-- Misc
DROP TABLE IF EXISTS np_claw_cards              CASCADE;
DROP TABLE IF EXISTS np_claw_projects           CASCADE;
DROP TABLE IF EXISTS np_claw_repo_monitors      CASCADE;
DROP TABLE IF EXISTS np_claw_pending_mutations  CASCADE;
DROP TABLE IF EXISTS np_claw_prompt_templates   CASCADE;
DROP TABLE IF EXISTS np_claw_research_queries   CASCADE;
DROP TABLE IF EXISTS np_claw_briefing_config    CASCADE;
DROP TABLE IF EXISTS np_claw_redfin_listings    CASCADE;
DROP TABLE IF EXISTS np_claw_instance_tables    CASCADE;

-- Link tables
DROP TABLE IF EXISTS np_mux_claw_threads        CASCADE;

DROP SCHEMA IF EXISTS np_claw CASCADE;

DELETE FROM _schema_version WHERE migration_name LIKE 'claw_%';

COMMIT;
