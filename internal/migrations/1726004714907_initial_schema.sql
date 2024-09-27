-- migration 1726004714907 initial_schema
---- disable-tx

CREATE TABLE IF NOT EXISTS migrations (
  id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  version BIGINT NOT NULL,
  name VARCHAR(4095) NOT NULL,
  namespace VARCHAR(4095) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE (version, namespace),
  CONSTRAINT version CHECK (version >= 1)
);

---- create above / drop below ----

DROP TABLE IF EXISTS migrations;
