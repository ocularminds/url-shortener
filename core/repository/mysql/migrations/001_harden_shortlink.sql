-- Back up the database before applying this one-time migration to a table
-- created by the original shortner.sql schema.
ALTER TABLE ShortLink
  MODIFY Shortened CHAR(8) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  MODIFY Original VARCHAR(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  MODIFY Expiry SMALLINT UNSIGNED NOT NULL DEFAULT 30,
  MODIFY Created DATETIME(6) NOT NULL,
  MODIFY Hits BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD INDEX idx_shortlink_original (Original(191)),
  ADD CONSTRAINT chk_shortlink_expiry CHECK (Expiry BETWEEN 1 AND 3650);
