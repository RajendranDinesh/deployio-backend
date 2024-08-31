-- Drop triggers and functions
DROP TRIGGER IF EXISTS set_updated_at_timestamp ON "deploy-io".projects;
DROP TRIGGER IF EXISTS prevent_projects_update ON "deploy-io".projects;
DROP TRIGGER IF EXISTS set_updated_at_timestamp ON "deploy-io".environments;
DROP TRIGGER IF EXISTS prevent_environments_update ON "deploy-io".environments;
DROP TRIGGER IF EXISTS set_updated_at_timestamp ON "deploy-io".builds;
DROP TRIGGER IF EXISTS prevent_builds_update ON "deploy-io".builds;
DROP TRIGGER IF EXISTS set_updated_at_timestamp ON "deploy-io".deployments;
DROP TRIGGER IF EXISTS prevent_deployments_update ON "deploy-io".deployments;

DROP FUNCTION IF EXISTS "deploy-io".update_timestamp_column();
DROP FUNCTION IF EXISTS "deploy-io".prevent_projects_update();
DROP FUNCTION IF EXISTS "deploy-io".prevent_environments_update();
DROP FUNCTION IF EXISTS "deploy-io".prevent_builds_update();
DROP FUNCTION IF EXISTS "deploy-io".prevent_deployments_update();

-- Drop tables
DROP TABLE IF EXISTS "deploy-io".deployments;
DROP TABLE IF EXISTS "deploy-io".builds;
DROP TABLE IF EXISTS "deploy-io".environments;
DROP TABLE IF EXISTS "deploy-io".projects;
DROP TABLE IF EXISTS "deploy-io".users;

-- Drop types
DROP TYPE IF EXISTS build_status;
DROP TYPE IF EXISTS triggered_by;

-- Drop schema
DROP SCHEMA IF EXISTS "deploy-io" CASCADE;
