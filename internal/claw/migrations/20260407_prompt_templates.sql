-- Sprint 05: Prompt Library & Templates

CREATE TABLE IF NOT EXISTS np_claw_prompt_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID,                      -- NULL for built-in prompts
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL,
  prompt_text TEXT NOT NULL,
  variables JSONB NOT NULL DEFAULT '[]',
  chain_next UUID REFERENCES np_claw_prompt_templates(id) ON DELETE SET NULL,
  is_builtin BOOLEAN NOT NULL DEFAULT false,
  usage_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_prompt_templates_category ON np_claw_prompt_templates(category);
CREATE INDEX IF NOT EXISTS idx_np_claw_prompt_templates_user_id ON np_claw_prompt_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_prompt_templates_builtin ON np_claw_prompt_templates(is_builtin);

-- Seed built-in prompts: Business (5)
INSERT INTO np_claw_prompt_templates (title, description, category, prompt_text, variables, is_builtin) VALUES
('Business Plan Outline', 'Generate a structured business plan outline for a new venture.', 'business',
 'Create a detailed business plan outline for {{business_name}} in the {{industry}} industry. Include executive summary, market analysis, competitive landscape, revenue model, and 3-year financial projections.',
 '[{"name": "business_name", "label": "Business Name", "type": "text"}, {"name": "industry", "label": "Industry", "type": "text"}]'::jsonb, true),

('SWOT Analysis', 'Perform a SWOT analysis for a company or product.', 'business',
 'Perform a detailed SWOT analysis for {{subject}}. For each quadrant (Strengths, Weaknesses, Opportunities, Threats), provide at least 5 items with brief explanations.',
 '[{"name": "subject", "label": "Company or Product", "type": "text"}]'::jsonb, true),

('Meeting Summary', 'Summarize meeting notes into action items and decisions.', 'business',
 'Summarize the following meeting notes into: 1) Key decisions made, 2) Action items with owners, 3) Open questions, 4) Next steps.\n\nMeeting notes:\n{{notes}}',
 '[{"name": "notes", "label": "Meeting Notes", "type": "textarea"}]'::jsonb, true),

('Elevator Pitch', 'Craft a concise elevator pitch for your idea.', 'business',
 'Write a 30-second elevator pitch for {{product}} targeting {{audience}}. Make it compelling, clear, and end with a call to action.',
 '[{"name": "product", "label": "Product/Service", "type": "text"}, {"name": "audience", "label": "Target Audience", "type": "text"}]'::jsonb, true),

('Competitive Analysis', 'Compare your product against competitors.', 'business',
 'Create a competitive analysis comparing {{our_product}} against {{competitors}}. Include feature comparison, pricing, market positioning, strengths and weaknesses of each.',
 '[{"name": "our_product", "label": "Your Product", "type": "text"}, {"name": "competitors", "label": "Competitors (comma-separated)", "type": "text"}]'::jsonb, true)
ON CONFLICT DO NOTHING;

-- Seed built-in prompts: Writing (5)
INSERT INTO np_claw_prompt_templates (title, description, category, prompt_text, variables, is_builtin) VALUES
('Blog Post Draft', 'Write a structured blog post on any topic.', 'writing',
 'Write a {{length}}-word blog post about "{{topic}}". Use a {{tone}} tone. Include an engaging introduction, 3-5 main sections with headers, and a conclusion with a call to action.',
 '[{"name": "topic", "label": "Topic", "type": "text"}, {"name": "length", "label": "Word Count", "type": "text", "default": "1000"}, {"name": "tone", "label": "Tone", "type": "select", "options": ["professional", "casual", "technical", "conversational"]}]'::jsonb, true),

('Email Draft', 'Compose a professional email.', 'writing',
 'Write a {{tone}} email to {{recipient}} about {{subject}}. Context: {{context}}',
 '[{"name": "recipient", "label": "Recipient", "type": "text"}, {"name": "subject", "label": "Subject", "type": "text"}, {"name": "context", "label": "Context", "type": "textarea"}, {"name": "tone", "label": "Tone", "type": "select", "options": ["formal", "friendly", "urgent", "apologetic"]}]'::jsonb, true),

('Content Rewriter', 'Rewrite content in a different style or for a different audience.', 'writing',
 'Rewrite the following text for a {{audience}} audience in a {{style}} style. Preserve the core meaning but adapt the language and complexity.\n\nOriginal:\n{{text}}',
 '[{"name": "text", "label": "Original Text", "type": "textarea"}, {"name": "audience", "label": "Target Audience", "type": "text"}, {"name": "style", "label": "Style", "type": "select", "options": ["simple", "technical", "academic", "marketing"]}]'::jsonb, true),

('Social Media Post', 'Generate social media content for any platform.', 'writing',
 'Write a {{platform}} post about {{topic}}. Make it engaging, include relevant hashtags, and keep it within platform character limits. Tone: {{tone}}.',
 '[{"name": "platform", "label": "Platform", "type": "select", "options": ["Twitter/X", "LinkedIn", "Instagram", "Facebook"]}, {"name": "topic", "label": "Topic", "type": "text"}, {"name": "tone", "label": "Tone", "type": "select", "options": ["professional", "casual", "humorous", "inspirational"]}]'::jsonb, true),

('Proofreader', 'Check text for grammar, style, and clarity issues.', 'writing',
 'Proofread the following text. Fix grammar, spelling, and punctuation errors. Suggest improvements for clarity and flow. Show changes with explanations.\n\n{{text}}',
 '[{"name": "text", "label": "Text to Proofread", "type": "textarea"}]'::jsonb, true)
ON CONFLICT DO NOTHING;

-- Seed built-in prompts: Coding (5)
INSERT INTO np_claw_prompt_templates (title, description, category, prompt_text, variables, is_builtin) VALUES
('Code Review', 'Get a thorough code review with suggestions.', 'coding',
 'Review the following {{language}} code. Check for bugs, security issues, performance problems, and style improvements. Provide specific suggestions.\n\n```{{language}}\n{{code}}\n```',
 '[{"name": "language", "label": "Language", "type": "text"}, {"name": "code", "label": "Code", "type": "textarea"}]'::jsonb, true),

('Explain Code', 'Get a clear explanation of what code does.', 'coding',
 'Explain the following {{language}} code step by step. Assume the reader has {{level}} experience. Include what each section does and why.\n\n```{{language}}\n{{code}}\n```',
 '[{"name": "language", "label": "Language", "type": "text"}, {"name": "code", "label": "Code", "type": "textarea"}, {"name": "level", "label": "Experience Level", "type": "select", "options": ["beginner", "intermediate", "advanced"]}]'::jsonb, true),

('Write Tests', 'Generate test cases for your code.', 'coding',
 'Write comprehensive tests for the following {{language}} code using {{framework}}. Include unit tests for happy paths, edge cases, and error conditions.\n\n```{{language}}\n{{code}}\n```',
 '[{"name": "language", "label": "Language", "type": "text"}, {"name": "framework", "label": "Test Framework", "type": "text"}, {"name": "code", "label": "Code", "type": "textarea"}]'::jsonb, true),

('Debug Helper', 'Help diagnose and fix a bug.', 'coding',
 'I have a bug in my {{language}} code. Expected behavior: {{expected}}. Actual behavior: {{actual}}. Help me find and fix the issue.\n\n```{{language}}\n{{code}}\n```',
 '[{"name": "language", "label": "Language", "type": "text"}, {"name": "expected", "label": "Expected Behavior", "type": "text"}, {"name": "actual", "label": "Actual Behavior", "type": "text"}, {"name": "code", "label": "Code", "type": "textarea"}]'::jsonb, true),

('API Design', 'Design a REST API for a given resource.', 'coding',
 'Design a RESTful API for managing {{resource}} in a {{context}} application. Include endpoints, request/response schemas (JSON), status codes, authentication requirements, and pagination.',
 '[{"name": "resource", "label": "Resource Name", "type": "text"}, {"name": "context", "label": "Application Context", "type": "text"}]'::jsonb, true)
ON CONFLICT DO NOTHING;

-- Seed built-in prompts: Research (4)
INSERT INTO np_claw_prompt_templates (title, description, category, prompt_text, variables, is_builtin) VALUES
('Topic Deep Dive', 'Get a comprehensive overview of any topic.', 'research',
 'Provide a comprehensive overview of {{topic}}. Cover: definition, history, current state, key players, recent developments, controversies, and future outlook. Cite specific examples where possible.',
 '[{"name": "topic", "label": "Topic", "type": "text"}]'::jsonb, true),

('Compare and Contrast', 'Compare two or more options objectively.', 'research',
 'Compare and contrast {{option_a}} vs {{option_b}}. Create a structured comparison covering: features, pros/cons, pricing, use cases, and a recommendation based on different scenarios.',
 '[{"name": "option_a", "label": "Option A", "type": "text"}, {"name": "option_b", "label": "Option B", "type": "text"}]'::jsonb, true),

('Summarize Document', 'Summarize a long document into key points.', 'research',
 'Summarize the following document into: 1) One-paragraph summary, 2) Key points (bullet list), 3) Important data/statistics, 4) Conclusions and recommendations.\n\n{{document}}',
 '[{"name": "document", "label": "Document Text", "type": "textarea"}]'::jsonb, true),

('Fact Check', 'Verify claims and provide sources.', 'research',
 'Fact-check the following claim: "{{claim}}". Provide: 1) Verdict (true/false/partially true/unverifiable), 2) Evidence for and against, 3) Context and nuance, 4) Related facts.',
 '[{"name": "claim", "label": "Claim to Verify", "type": "textarea"}]'::jsonb, true)
ON CONFLICT DO NOTHING;

-- Seed built-in prompts: Creative (4)
INSERT INTO np_claw_prompt_templates (title, description, category, prompt_text, variables, is_builtin) VALUES
('Story Generator', 'Generate a short story from a premise.', 'creative',
 'Write a {{length}} short story in the {{genre}} genre. Premise: {{premise}}. Include vivid descriptions, dialogue, and a satisfying ending.',
 '[{"name": "premise", "label": "Story Premise", "type": "textarea"}, {"name": "genre", "label": "Genre", "type": "select", "options": ["sci-fi", "fantasy", "mystery", "romance", "horror", "literary fiction"]}, {"name": "length", "label": "Length", "type": "select", "options": ["500-word", "1000-word", "2000-word"]}]'::jsonb, true),

('Brainstorm Ideas', 'Generate creative ideas for any challenge.', 'creative',
 'Generate {{count}} creative ideas for {{challenge}}. For each idea, provide: a catchy name, one-sentence description, and why it could work. Think outside the box.',
 '[{"name": "challenge", "label": "Challenge/Problem", "type": "textarea"}, {"name": "count", "label": "Number of Ideas", "type": "text", "default": "10"}]'::jsonb, true),

('Character Creator', 'Create a detailed fictional character.', 'creative',
 'Create a detailed character profile for a {{role}} in a {{setting}} setting. Include: name, age, appearance, personality traits, backstory, motivations, flaws, and a memorable quirk.',
 '[{"name": "role", "label": "Character Role", "type": "text"}, {"name": "setting", "label": "Setting/World", "type": "text"}]'::jsonb, true),

('Dialogue Writer', 'Write realistic dialogue between characters.', 'creative',
 'Write a dialogue scene between {{characters}} about {{situation}}. Make each voice distinct. Include stage directions and emotional beats. Tone: {{tone}}.',
 '[{"name": "characters", "label": "Characters (e.g. a detective and a suspect)", "type": "text"}, {"name": "situation", "label": "Situation", "type": "text"}, {"name": "tone", "label": "Tone", "type": "select", "options": ["dramatic", "comedic", "tense", "heartfelt"]}]'::jsonb, true)
ON CONFLICT DO NOTHING;
