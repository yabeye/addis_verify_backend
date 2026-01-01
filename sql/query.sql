/***** ACCOUNTS *****/

-- name: UpsertAccount :one
-- This is used ONLY during the VerifyOTP flow.
-- It moves the 'token_valid_from' forward to invalidate old sessions.
INSERT INTO accounts (phone, token_valid_from)
VALUES ($1, NOW())
ON CONFLICT (phone) DO UPDATE 
SET 
    token_valid_from = EXCLUDED.token_valid_from,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpdateAccountStatus :exec
-- This is for administrative or system changes. 
-- It does NOT touch token_valid_from, so the user stays logged in.
UPDATE accounts 
SET status = $2, updated_at = CURRENT_TIMESTAMP 
WHERE id = $1;

-- name: GetAccountByPhone :one
SELECT * FROM accounts WHERE phone = $1 LIMIT 1;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1 LIMIT 1;