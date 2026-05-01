-- Replication user creation.
-- Runs ONLY on the primary (mounted only there in compose).
-- The replica connects to primary using these credentials to pull binlog events.
--
-- Production note: hardcoded password is fine for a learning project, but real
-- production would use Docker secrets / Vault / SSM Parameter Store.

CREATE OR REPLACE USER 'replicator'@'%' IDENTIFIED BY 'replicapass';
GRANT REPLICATION SLAVE ON *.* TO 'replicator'@'%';
FLUSH PRIVILEGES;
