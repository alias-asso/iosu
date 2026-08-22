-- name: GetSiteConfig :one
SELECT * FROM site_config WHERE id = 1;

-- name: CreateSiteConfigIfMissing :execrows
INSERT OR IGNORE INTO site_config
    (id, site_title, main_text, secondary_text, current_contest,
     help_content, rules_content, legal_content, credits_content)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateSiteConfig :exec
UPDATE site_config SET
    site_title      = COALESCE(sqlc.narg('site_title'), site_title),
    main_text       = COALESCE(sqlc.narg('main_text'), main_text),
    secondary_text  = COALESCE(sqlc.narg('secondary_text'), secondary_text),
    current_contest = COALESCE(sqlc.narg('current_contest'), current_contest),
    help_content    = COALESCE(sqlc.narg('help_content'), help_content),
    rules_content   = COALESCE(sqlc.narg('rules_content'), rules_content),
    legal_content   = COALESCE(sqlc.narg('legal_content'), legal_content),
    credits_content = COALESCE(sqlc.narg('credits_content'), credits_content)
WHERE id = 1;
