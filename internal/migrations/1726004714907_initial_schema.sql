-- migration 1726004714907 initial_schema

CREATE TABLE IF NOT EXISTS conduitmigrations (
  id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  version VARCHAR(255) NOT NULL,
  name VARCHAR(4095) NOT NULL,
  namespace VARCHAR(4095) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE (version, namespace),
  CONSTRAINT version CHECK (version >= 1)
);
