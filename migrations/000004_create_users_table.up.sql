CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
create extension citext;
CREATE TABLE IF NOT EXISTS users (
                                     id bigserial PRIMARY KEY,
                                     created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
                                     name text NOT NULL,
                                     email citext UNIQUE NOT NULL,
                                     password_hash bytea NOT NULL,
                                     activated bool NOT NULL,
                                     version uuid NOT NULL DEFAULT uuid_generate_v4()
);
