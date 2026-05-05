-- Sprint 08: Business/Solopreneur Agent — prompt templates + proactive job templates

-- Add persona_name column to prompt_templates for persona-specific filtering
ALTER TABLE np_claw_prompt_templates ADD COLUMN IF NOT EXISTS persona_name TEXT;
CREATE INDEX IF NOT EXISTS idx_np_claw_prompt_templates_persona ON np_claw_prompt_templates(persona_name);

-- Business-specific prompt templates (linked to BusinessStrategist persona)
INSERT INTO np_claw_prompt_templates (title, description, category, prompt_text, variables, is_builtin, persona_name) VALUES
('Revenue Analysis', 'Analyze income and expenses from budget data to identify trends and opportunities.', 'business',
 'Analyze my business finances for {{time_period}}. Pull data from the budget plugin and provide: 1) Revenue breakdown by source, 2) Top expense categories, 3) Profit margin trends, 4) Cash flow concerns, 5) Three actionable recommendations to improve profitability.',
 '[{"name": "time_period", "label": "Time Period", "type": "select", "options": ["this week", "this month", "last 30 days", "this quarter", "this year"]}]'::jsonb, true, 'BusinessStrategist'),

('Content Calendar', 'Plan a week of content across your marketing channels.', 'business',
 'Create a content calendar for {{platform}} for the next {{duration}}. My business is {{business_desc}}. Target audience: {{audience}}. Include: specific post topics, suggested posting times, content format (text/image/video), hashtags, and a mix of educational, promotional, and engagement content.',
 '[{"name": "platform", "label": "Platform", "type": "select", "options": ["Twitter/X", "LinkedIn", "Instagram", "All platforms"]}, {"name": "duration", "label": "Duration", "type": "select", "options": ["1 week", "2 weeks", "1 month"]}, {"name": "business_desc", "label": "Business Description", "type": "text"}, {"name": "audience", "label": "Target Audience", "type": "text"}]'::jsonb, true, 'BusinessStrategist'),

('Client Outreach', 'Draft personalized outreach emails to potential or existing clients.', 'business',
 'Draft a {{outreach_type}} email to {{recipient_type}}. My business offers {{offering}}. The goal is {{goal}}. Keep it under 200 words, personable but professional. Include a clear call to action. Draft the subject line too.',
 '[{"name": "outreach_type", "label": "Outreach Type", "type": "select", "options": ["cold outreach", "follow-up", "re-engagement", "upsell", "referral request"]}, {"name": "recipient_type", "label": "Recipient", "type": "text", "default": "a potential client"}, {"name": "offering", "label": "Your Offering", "type": "text"}, {"name": "goal", "label": "Email Goal", "type": "text"}]'::jsonb, true, 'BusinessStrategist'),

('Competitive Research', 'Research competitors and identify your positioning advantages.', 'business',
 'Research the competitive landscape for {{my_business}} in the {{market}} market. Identify the top 3-5 competitors. For each: pricing model, key features, strengths, weaknesses, and target audience. Then recommend how I should position against them and where the biggest opportunity gaps are.',
 '[{"name": "my_business", "label": "Your Business", "type": "text"}, {"name": "market", "label": "Market/Niche", "type": "text"}]'::jsonb, true, 'BusinessStrategist'),

('Weekly Review', 'Summarize the week''s business metrics and plan next actions.', 'business',
 'Generate my weekly business review. Pull available data and summarize: 1) Revenue this week vs last week, 2) New leads or clients, 3) Content performance highlights, 4) Tasks completed vs planned, 5) Top 3 priorities for next week, 6) Any risks or blockers to address. Keep it actionable and under 500 words.',
 '[]'::jsonb, true, 'BusinessStrategist')
ON CONFLICT DO NOTHING;

-- Proactive job templates for business persona
INSERT INTO np_claw_proactive_jobs (job_type, enabled, cron_expression, config, quiet_hours_start, quiet_hours_end) VALUES
('weekly_revenue_summary', false, '0 18 * * 0', '{"persona": "BusinessStrategist", "description": "Weekly revenue summary every Sunday at 6 PM UTC"}'::jsonb, 22, 7),
('daily_content_reminder', false, '0 8 * * 1-5', '{"persona": "BusinessStrategist", "description": "Daily content reminder on weekday mornings at 8 AM UTC"}'::jsonb, 22, 7),
('monthly_goal_review', false, '0 9 1 * *', '{"persona": "BusinessStrategist", "description": "Monthly goal review on the 1st at 9 AM UTC"}'::jsonb, 22, 7)
ON CONFLICT DO NOTHING;
