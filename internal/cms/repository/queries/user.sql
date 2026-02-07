-- name: CreateUser :one
INSERT INTO
    users (email, password_hash, role)
VALUES
    ($ 1, $ 2, $ 3) RETURNING *;

-- name: GetUserByID :one
SELECT
    *
FROM
    users
WHERE
    id = $ 1;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = $ 1;

-- Note: This is needed to determine if we should create the
--          default admin. It can be optimized by limiting the 
--          query to 1 (check if table is empty or not).
-- name: CountUsers :one
SELECT
    COUNT(*)
FROM
    users;