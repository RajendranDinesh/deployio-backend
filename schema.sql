CREATE SCHEMA "deploy-io";

CREATE OR REPLACE FUNCTION "deploy-io".update_timestamp_column()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = NOW();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- users
CREATE TABLE IF NOT EXISTS "deploy-io".users (
	id serial8 NOT NULL,
	email varchar NOT NULL,
	name varchar NOT NULL,
	status boolean DEFAULT true,
	"access" varchar NOT NULL,
	"refresh" varchar NOT NULL,
	access_expires_by TIMESTAMP NOT NULL,
	refresh_expires_by TIMESTAMP NOT NULL,
	CONSTRAINT users_pk PRIMARY KEY (id),
	CONSTRAINT users_unique UNIQUE (email)
);

-- projects
CREATE TABLE IF NOT EXISTS "deploy-io".projects (
	id serial8,
	user_id serial8 NOT NULL,
	name VARCHAR NOT NULL,
	github_id serial NOT NULL,
	directory VARCHAR DEFAULT './',
	node_version VARCHAR DEFAULT 20,
	install_command VARCHAR DEFAULT 'npm install',
	build_command VARCHAR DEFAULT 'npm run build',
	output_folder VARCHAR DEFAULT './build',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT projects_pk PRIMARY KEY(id),
	CONSTRAINT projects_unique_name UNIQUE(name),
	CONSTRAINT projects_unique_github_id UNIQUE(github_id),
	CONSTRAINT projects_fk FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TRIGGER set_updated_at_timestamp
BEFORE UPDATE ON "deploy-io".projects
FOR EACH ROW 
EXECUTE FUNCTION update_timestamp_column();

CREATE OR REPLACE FUNCTION "deploy-io".prevent_projects_update() 
RETURNS TRIGGER AS $$
BEGIN
	IF NEW.created_at IS DISTINCT FROM OLD.created_at THEN
    	RAISE EXCEPTION 'You cannot change the value.';
    END IF;
   
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_projects_update
BEFORE UPDATE ON "deploy-io".projects 
FOR EACH ROW
EXECUTE FUNCTION prevent_projects_update();

-- environments
CREATE TABLE IF NOT EXISTS "deploy-io".environments (
	project_id serial8,
	key VARCHAR NOT NULL,
	value VARCHAR NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT env_fk FOREIGN KEY (project_id) REFERENCES projects(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TRIGGER set_updated_at_timestamp
BEFORE UPDATE ON "deploy-io".environments
FOR EACH ROW 
EXECUTE FUNCTION update_timestamp_column();

CREATE OR REPLACE FUNCTION "deploy-io".prevent_environments_update() 
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.created_at IS DISTINCT FROM OLD.created_at THEN
    	RAISE EXCEPTION 'You cannot change the value.';
    END IF;
   
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_environments_update
BEFORE UPDATE ON "deploy-io".environments 
FOR EACH ROW
EXECUTE FUNCTION prevent_environments_update();

-- builds
CREATE TYPE build_status AS ENUM ('in queue', 'running', 'success', 'failure');
CREATE TYPE triggered_by AS ENUM ('push', 'manual');

CREATE TABLE IF NOT EXISTS "deploy-io".builds (
	id serial8,
	project_id serial8,
	status build_status NOT NULL,
	triggered_by triggered_by NOT NULL,
	commit_hash VARCHAR NOT NULL,
	logs TEXT,
	start_time TIMESTAMP NULL,
	end_time TIMESTAMP NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT builds_pk PRIMARY KEY (id),
	CONSTRAINT builds_fk FOREIGN KEY (project_id) REFERENCES projects(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TRIGGER set_updated_at_timestamp
BEFORE UPDATE ON "deploy-io".builds
FOR EACH ROW 
EXECUTE FUNCTION update_timestamp_column();

CREATE OR REPLACE FUNCTION "deploy-io".prevent_builds_update() 
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.created_at IS DISTINCT FROM OLD.created_at THEN
    	RAISE EXCEPTION 'You cannot change the value.';
    END IF;
   
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_builds_update
BEFORE UPDATE ON "deploy-io".builds 
FOR EACH ROW
EXECUTE FUNCTION prevent_builds_update();

-- deployments
CREATE TABLE IF NOT EXISTS "deploy-io".deployments (
	id serial8,
	project_id serial8 NOT NULL,
	build_id serial8 NOT NULL,
	status BOOL NOT NULL DEFAULT true,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT deployments_pk PRIMARY KEY (id),
	CONSTRAINT deployments_fk_project FOREIGN KEY (project_id) REFERENCES projects(id) ON UPDATE CASCADE ON DELETE CASCADE,
	CONSTRAINT deployments_fk_build FOREIGN KEY (build_id) REFERENCES builds(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TRIGGER set_updated_at_timestamp
BEFORE UPDATE ON "deploy-io".deployments
FOR EACH ROW 
EXECUTE FUNCTION update_timestamp_column();

CREATE OR REPLACE FUNCTION "deploy-io".prevent_deployments_update() 
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.created_at IS DISTINCT FROM OLD.created_at THEN
    	RAISE EXCEPTION 'You cannot change the value.';
    END IF;
   
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_deployments_update
BEFORE UPDATE ON "deploy-io".deployments 
FOR EACH ROW
EXECUTE FUNCTION prevent_deployments_update();
