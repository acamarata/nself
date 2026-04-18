-- RLS policy template for nSelf tables.
--
-- Hasura session variable pattern:
--   current_setting('hasura.user', true)  -- UUID of the authenticated user,
--                                            injected by Hasura JWT claims via
--                                            x-hasura-user-id. The second arg
--                                            (true) means the function returns
--                                            NULL instead of raising an error
--                                            when the setting is not set (e.g.
--                                            anonymous or service-role sessions).
--   current_setting('hasura.role', true)  -- Role string ('user', 'admin', etc.)
--                                            injected via x-hasura-role claim.
--                                            Admin bypass uses this to allow
--                                            privileged operations across all rows.
--
-- Replace :schema and :table with the target schema and table name before
-- running this template. These are quoted identifiers; do not use string
-- interpolation with user-supplied input.

-- 1. Enable row-level security.
ALTER TABLE ":schema".":table" ENABLE ROW LEVEL SECURITY;

-- 2. Force RLS even for table owners (prevents privilege escalation via role SET).
ALTER TABLE ":schema".":table" FORCE ROW LEVEL SECURITY;

-- 3. Drop pre-existing standard policies so this template is idempotent.
DROP POLICY IF EXISTS ":table_select_own" ON ":schema".":table";
DROP POLICY IF EXISTS ":table_insert_own" ON ":schema".":table";
DROP POLICY IF EXISTS ":table_update_own" ON ":schema".":table";
DROP POLICY IF EXISTS ":table_delete_own" ON ":schema".":table";

-- 4a. SELECT: own rows or admin bypass.
CREATE POLICY ":table_select_own"
    ON ":schema".":table"
    FOR SELECT
    USING (
        (current_setting('hasura.user', true))::uuid = user_id
        OR current_setting('hasura.role', true) = 'admin'
    );

-- 4b. INSERT: own rows or admin bypass.
CREATE POLICY ":table_insert_own"
    ON ":schema".":table"
    FOR INSERT
    WITH CHECK (
        (current_setting('hasura.user', true))::uuid = user_id
        OR current_setting('hasura.role', true) = 'admin'
    );

-- 4c. UPDATE: own rows or admin bypass (USING + WITH CHECK for full row control).
CREATE POLICY ":table_update_own"
    ON ":schema".":table"
    FOR UPDATE
    USING (
        (current_setting('hasura.user', true))::uuid = user_id
        OR current_setting('hasura.role', true) = 'admin'
    )
    WITH CHECK (
        (current_setting('hasura.user', true))::uuid = user_id
        OR current_setting('hasura.role', true) = 'admin'
    );

-- 4d. DELETE: own rows or admin bypass.
CREATE POLICY ":table_delete_own"
    ON ":schema".":table"
    FOR DELETE
    USING (
        (current_setting('hasura.user', true))::uuid = user_id
        OR current_setting('hasura.role', true) = 'admin'
    );
