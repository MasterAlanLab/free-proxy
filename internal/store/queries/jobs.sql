-- name: CreateJob :exec
INSERT INTO jobs (id, name, status, created_at) VALUES (?, ?, ?, ?);

-- name: GetJob :one
SELECT * FROM jobs WHERE id = ?;

-- name: MarkJobRunning :exec
UPDATE jobs SET status = 'running', started_at = ? WHERE id = ?;

-- name: MarkJobSucceeded :exec
UPDATE jobs SET status = 'succeeded', finished_at = ?, result = ? WHERE id = ?;

-- name: MarkJobFailed :exec
UPDATE jobs SET status = 'failed', finished_at = ?, error = ? WHERE id = ?;

-- name: CancelUnfinishedJobs :exec
UPDATE jobs SET status = 'cancelled', finished_at = ?
WHERE status IN ('pending', 'running');

-- name: DeleteOldJobs :exec
DELETE FROM jobs WHERE created_at < ?;
