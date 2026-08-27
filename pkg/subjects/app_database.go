package subjects

const (
	RuntimeAppDBResolveQuerySubject           = "runtime.v1.query.app.db.resolve"
	RuntimeAppDBResolveQueueGroup             = "app-runtime-db-resolvers"
	RuntimeAppDBPurgeRowsCommandSubject       = "runtime.v1.cmd.app.db.purge-rows"
	RuntimeAppDBPurgeRowsQueueGroup           = "app-runtime-db-purgers"
	RuntimeAppDBCleanupPolicyQuerySubject     = "runtime.v1.query.app.db.cleanup-policy"
	RuntimeAppDBCleanupPolicyQueryQueueGroup  = "app-runtime-db-cleanup-policy-readers"
	RuntimeAppDBCleanupPolicyUpdateSubject    = "runtime.v1.cmd.app.db.cleanup-policy.update"
	RuntimeAppDBCleanupPolicyUpdateQueueGroup = "app-runtime-db-cleanup-policy-updaters"
)
