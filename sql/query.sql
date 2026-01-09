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



/***** USERS & ADDRESS *****/

-- name: GetUserWithAddressByAccountID :one
-- Retrieves the full user profile along with their primary address via JOIN.
SELECT 
    u.id as user_id, u.first_name, u.middle_name, u.last_name, u.alias_name, 
    u.birthdate, u.gender, u.citizenship, u.email, 
    u.user_head_shot_image, u.government_id_image, u.passport_image,
    a.id as address_id, a.country, a.region, a.city, a.zone, a.wereda, a.kebele
FROM users u
LEFT JOIN address a ON u.account_id = a.account_id
WHERE u.account_id = $1 LIMIT 1;

-- name: UpsertUser :one
-- Creates or updates the user profile linked to an account.
INSERT INTO users (
    account_id, first_name, middle_name, last_name, alias_name, 
    birthdate, gender, citizenship, email, 
    user_head_shot_image, government_id_image, passport_image
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (account_id) DO UPDATE SET
    -- Text fields protected: if EXCLUDED is '', keep the current value
    first_name = COALESCE(NULLIF(EXCLUDED.first_name, ''), users.first_name),
    middle_name = COALESCE(NULLIF(EXCLUDED.middle_name, ''), users.middle_name),
    last_name = COALESCE(NULLIF(EXCLUDED.last_name, ''), users.last_name),
    alias_name = COALESCE(NULLIF(EXCLUDED.alias_name, ''), users.alias_name),
    
    -- Date and Choice fields protected:
    birthdate = COALESCE(EXCLUDED.birthdate, users.birthdate),
    gender = COALESCE(NULLIF(EXCLUDED.gender, ''), users.gender),
    citizenship = COALESCE(NULLIF(EXCLUDED.citizenship, ''), users.citizenship),
    email = COALESCE(NULLIF(EXCLUDED.email, ''), users.email),
    
    -- Image fields (already protected in your previous version)
    user_head_shot_image = COALESCE(NULLIF(EXCLUDED.user_head_shot_image, ''), users.user_head_shot_image),
    government_id_image = COALESCE(NULLIF(EXCLUDED.government_id_image, ''), users.government_id_image),
    passport_image = COALESCE(NULLIF(EXCLUDED.passport_image, ''), users.passport_image),
    
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpsertAddress :one
-- Creates or updates the address linked to an account.
INSERT INTO address (
    account_id, country, region, city, zone, wereda, kebele
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (account_id) DO UPDATE SET
    country = COALESCE(NULLIF(EXCLUDED.country, ''), address.country),
    region  = COALESCE(NULLIF(EXCLUDED.region, ''), address.region),
    city    = COALESCE(NULLIF(EXCLUDED.city, ''), address.city),
    zone    = COALESCE(NULLIF(EXCLUDED.zone, ''), address.zone),
    wereda  = COALESCE(NULLIF(EXCLUDED.wereda, ''), address.wereda),
    kebele  = COALESCE(NULLIF(EXCLUDED.kebele, ''), address.kebele),
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpdateUserImages :exec
-- Specifically for updating document or profile images.
UPDATE users 
SET 
    user_head_shot_image = COALESCE($2, user_head_shot_image),
    government_id_image = COALESCE($3, government_id_image),
    passport_image = COALESCE($4, passport_image),
    updated_at = CURRENT_TIMESTAMP
WHERE account_id = $1;