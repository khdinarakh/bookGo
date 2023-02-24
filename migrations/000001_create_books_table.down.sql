CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS books (
                                      id bigserial PRIMARY KEY,
                                      created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
                                      title text NOT NULL,
                                      content text NOT NULL,
                                      year integer NOT NULL,
                                      pages integer NOT NULL,
                                      genres text[] NOT NULL,
                                      version uuid NOT NULL DEFAULT uuid_generate_v4()
);