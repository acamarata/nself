-- Rollback for 20260417_p93_rls_all_tables.sql
-- Removes user_isolation policy and disables RLS on all 72 tables covered by S02-T01.
-- Tables covered by earlier migrations are left unchanged by this rollback.

DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'np_claw_actions',
        'np_claw_admin_context',
        'np_claw_agent_queue',
        'np_claw_agent_tasks',
        'np_claw_agents',
        'np_claw_ai_usage',
        'np_claw_api_keys',
        'np_claw_api_usage',
        'np_claw_attachments',
        'np_claw_audit_export_rate',
        'np_claw_audit_log',
        'np_claw_audit_retention_config',
        'np_claw_cards',
        'np_claw_config',
        'np_claw_cv_data',
        'np_claw_domain_expert_config',
        'np_claw_habit_logs',
        'np_claw_habits',
        'np_claw_health_checkins',
        'np_claw_image_jobs',
        'np_claw_knowledge_edges',
        'np_claw_knowledge_nodes',
        'np_claw_marketplace_installs',
        'np_claw_marketplace_ratings',
        'np_claw_media_embeddings',
        'np_claw_media_events',
        'np_claw_memories',
        'np_claw_memory',
        'np_claw_memory_history',
        'np_claw_memory_links',
        'np_claw_memory_rooms',
        'np_claw_message_summaries',
        'np_claw_message_topics',
        'np_claw_monitor_events',
        'np_claw_monitors',
        'np_claw_pdf_pages',
        'np_claw_pdf_toc',
        'np_claw_pending_mutations',
        'np_claw_personas',
        'np_claw_playbooks',
        'np_claw_proactive_jobs',
        'np_claw_profile_ab_tests',
        'np_claw_profile_quality_daily',
        'np_claw_project_files',
        'np_claw_projects',
        'np_claw_prompt_templates',
        'np_claw_prune_policy',
        'np_claw_reflections',
        'np_claw_research_results',
        'np_claw_retrieval_profiles',
        'np_claw_routing_rules',
        'np_claw_saved_searches',
        'np_claw_sessions',
        'np_claw_share_comments',
        'np_claw_shared_sessions',
        'np_claw_skill_installs',
        'np_claw_skill_listings',
        'np_claw_skill_ratings',
        'np_claw_system_prompts',
        'np_claw_team_members',
        'np_claw_teams',
        'np_claw_tools',
        'np_claw_topic_knowledge',
        'np_claw_topic_prefs',
        'np_claw_topic_relations',
        'np_claw_topic_transitions',
        'np_claw_topics',
        'np_claw_transcripts',
        'np_claw_tutor_cards',
        'np_claw_user_keys',
        'np_claw_user_preferences',
        'np_claw_voice_calls',
        'np_claw_webhook_dlq'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = t) THEN
            BEGIN
                EXECUTE format('DROP POLICY IF EXISTS user_isolation ON %I', t);
                EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', t);
                EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
            EXCEPTION WHEN OTHERS THEN
                RAISE WARNING 'S02-T01 rollback: skipped table % — %: %', t, SQLSTATE, SQLERRM;
            END;
        END IF;
    END LOOP;
END $$;
