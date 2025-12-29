/*****ACCOUNTS*****/


-- name: UpsertAccount :one
INSERT INTO accounts (phone)
VALUES ($1)
ON CONFLICT (phone) DO UPDATE 
SET updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpdateAccountStatus :exec
UPDATE accounts 
SET status = $2, updated_at = CURRENT_TIMESTAMP 
WHERE id = $1;

-- name: GetAccountByPhone :one
SELECT * FROM accounts
WHERE phone = $1 LIMIT 1;