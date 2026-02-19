-- name: CreateUser :one
insert into users (id, created_at, updated_at, email, hashed_password)
values (
    gen_random_uuid(),
    now(),
    now(),
    $1,
    $2
)
returning *;

-- name: DeleteUsers :exec
delete from users;

-- name: GetUserByEmail :one
select id, created_at, updated_at, email, hashed_password, is_chirpy_red
from users
where email = $1;

-- name: UpdateUserByID :one
update users
set email = $1, hashed_password = $2, updated_at = now()
where id = $3
returning *;

-- name: UpdateUserChirpyRedByID :one
update users
set is_chirpy_red = true
where id = $1
returning *;