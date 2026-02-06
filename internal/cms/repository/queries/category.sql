-- name: CreateCategory :one
INSERT INTO categories (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetCategoryByID :one
SELECT * FROM categories WHERE id = $1;

-- name: GetCategoryByName :one
SELECT * FROM categories WHERE name = $1;

-- name: ListCategories :many
SELECT * FROM categories ORDER BY name;
