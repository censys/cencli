-- name: InsertGlobal :one
INSERT INTO
    globals (name, description, value, created_at, last_used_at)
VALUES
    (?, ?, ?, ?, ?)
RETURNING
    id;

-- name: GetGlobalsByName :many
SELECT
    *
FROM
    globals
WHERE
    name = ?
ORDER BY
    id ASC;

-- name: DeleteGlobal :one
DELETE FROM
    globals
WHERE
    id = ?
RETURNING
    *;

-- name: UpdateGlobalLastUsedAt :one
UPDATE
    globals
SET
    last_used_at = ?
WHERE
    id = ?
RETURNING
    *;

-- name: GetLastUsedGlobalByName :one
SELECT
    *
FROM
    globals
WHERE
    name = ?
ORDER BY
    last_used_at DESC
LIMIT 1;
