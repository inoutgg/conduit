-- migration: 20250629171951_initial_schema.sql

CREATE TABLE IF NOT EXISTS conduitmigrations (
  id BIGSERIAL NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  version VARCHAR(255) NOT NULL,
  name VARCHAR(4095) NOT NULL,
  namespace VARCHAR(4095) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE (version, namespace)
);
