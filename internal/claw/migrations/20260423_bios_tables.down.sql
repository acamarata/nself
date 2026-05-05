-- Rollback: Claw BIOS tables (SP-20 C17a)
DROP TABLE IF EXISTS np_claw_bios_snapshots;
DROP TABLE IF EXISTS np_claw_bios_config;
DROP TABLE IF EXISTS np_claw_tool_registry;
