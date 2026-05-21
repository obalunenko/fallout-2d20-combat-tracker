-- +goose Up
ALTER TABLE campaigns ADD COLUMN start_date_datetime DATETIME NOT NULL DEFAULT '1970-01-01 00:00:00.000';

UPDATE campaigns
SET start_date_datetime = COALESCE(
    STRFTIME(
        '%Y-%m-%d %H:%M:%f',
        CASE
            WHEN length(trim(start_date)) = 10 THEN trim(start_date) || ' 00:00:00'
            ELSE trim(start_date)
        END
    ),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
);

ALTER TABLE campaigns DROP COLUMN start_date;
ALTER TABLE campaigns RENAME COLUMN start_date_datetime TO start_date;

-- +goose Down
ALTER TABLE campaigns ADD COLUMN start_date_text TEXT NOT NULL DEFAULT '1970-01-01';

UPDATE campaigns
SET start_date_text = COALESCE(DATE(start_date), substr(start_date, 1, 10), DATE('now'));

ALTER TABLE campaigns DROP COLUMN start_date;
ALTER TABLE campaigns RENAME COLUMN start_date_text TO start_date;
