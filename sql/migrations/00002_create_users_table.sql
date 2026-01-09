-- +goose Up
-- +goose StatementBegin

-- 1. Create table 'users'
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    last_name VARCHAR(100) NOT NULL,
    alias_name VARCHAR(100),
    birthdate DATE,
    gender VARCHAR(20),
    citizenship VARCHAR(100),
    email VARCHAR(255) UNIQUE,
    user_head_shot_image TEXT, -- Storing the URL/Path to the image
    government_id_image TEXT,
    passport_image TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign Key Constraint
    CONSTRAINT fk_account FOREIGN KEY(account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

-- 2. Create table 'address'
CREATE TABLE IF NOT EXISTS address (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL,
    country VARCHAR(100) NOT NULL DEFAULT 'Ethiopia',
    region VARCHAR(100),
    city VARCHAR(100),
    zone VARCHAR(100),
    wereda VARCHAR(100),
    kebele VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign Key Constraint
    CONSTRAINT fk_account_address FOREIGN KEY(account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

-- 3. Add triggers for automatic updated_at (Assuming you have a function called update_timestamp)
-- CREATE TRIGGER set_timestamp_users BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE update_timestamp();
-- CREATE TRIGGER set_timestamp_address BEFORE UPDATE ON address FOR EACH ROW EXECUTE PROCEDURE update_timestamp();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS address;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd