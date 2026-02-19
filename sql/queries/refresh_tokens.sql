-- name: CreateRefreshToken :one
insert into refresh_tokens (
    token,
    created_at,
    updated_at,
    user_id,
    expires_at,
    revoked_at
)
values (
    $1,
    now(),
    now(),
    $2,
    $3,
    null
)
returning *;

-- name: GetRefreshToken :one
select token, created_at, updated_at, user_id, expires_at, revoked_at
from refresh_tokens
where token = $1;

-- name: RevokeRefreshToken :exec
update refresh_tokens
set revoked_at = $1, updated_at = $1
where token = $2;