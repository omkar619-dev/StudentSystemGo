-- Replication setup script.
-- Runs ONLY on the replica (mounted only there in compose).
-- Tells this server to pull binlog events from `mysql_primary` and apply them.
--
-- This file runs on the FIRST boot only (when the replica's data dir is empty).
-- On subsequent restarts, replication state persists in the replica's data dir
-- and resumes automatically.

-- ── Wipe replica's local binlog/GTID state ─────────────────
-- During replica's startup, MariaDB's docker-entrypoint creates its own local
-- transactions (system tables, schooldb database, etc.) — these get GTID 0-2-N
-- on this replica. When we then try to pull primary's GTID 0-1-N, gtid_strict_mode
-- detects out-of-order sequence numbers and refuses.
--
-- RESET MASTER wipes this replica's own binlog history and GTID counter back to 0.
-- After this, the replica is "blank" from a GTID perspective and can cleanly
-- accept primary's full binlog history starting from primary's GTID 0-1-1.
RESET MASTER;

-- ── Configure replication ──────────────────────────────────
CHANGE MASTER TO
  MASTER_HOST='mysql_primary',          -- compose service name → resolves via Docker DNS
  MASTER_PORT=3306,
  MASTER_USER='replicator',             -- created on primary by 00-replication-user.sql
  MASTER_PASSWORD='replicapass',
  MASTER_USE_GTID=slave_pos,            -- track progress by GTID, not filename+pos
  MASTER_CONNECT_RETRY=10;              -- retry every 10s if disconnected

START SLAVE;
