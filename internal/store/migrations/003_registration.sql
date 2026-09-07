ALTER TABLE site_config ADD COLUMN registration_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE site_config ADD COLUMN registration_requires_approval BOOLEAN NOT NULL DEFAULT FALSE;
