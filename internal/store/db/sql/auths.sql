-- name: InsertAuth :one
INSERT INTO
    auths (name, description, value, created_at, last_used_at)
VALUES
    (?, ?, ?, ?, ?)
RETURNING
    id;

-- name: GetAuthsByName :many
SELECT
    *
FROM
    auths
WHERE
    name = ?
ORDER BY
    id ASC;

-- name: DeleteAuth :one
DELETE FROM
    auths
WHERE
    id = ?
RETURNING
    *;

-- name: UpdateAuthLastUsedAt :one
UPDATE
    auths
SET
    last_used_at = ?
WHERE
    id = ?
RETURNING
    *;

-- name: GetLastUsedAuthByName :one
SELECT
    *
FROM
    auths
WHERE
    name = ?
ORDER BY
    last_used_at DESC
LIMIT 1;
