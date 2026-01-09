-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD CONSTRAINT users_account_id_unique UNIQUE (account_id);
ALTER TABLE address ADD CONSTRAINT address_account_id_unique UNIQUE (account_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT users_account_id_unique;
ALTER TABLE address DROP CONSTRAINT address_account_id_unique;
-- +goose StatementEnd