package app

import (
	"strings"

	"gorm.io/gorm"
)

type DBDialect string

const (
	DBDialectSQLite  DBDialect = "sqlite"
	DBDialectMySQL   DBDialect = "mysql"
	DBDialectUnknown DBDialect = "unknown"
)

type TimeBucket string

const (
	TimeBucketHour  TimeBucket = "hour"
	TimeBucketDay   TimeBucket = "day"
	TimeBucketMonth TimeBucket = "month"
)

// DetectDBDialect returns the normalized dialect name for the current GORM DB.
func DetectDBDialect(db *gorm.DB) DBDialect {
	if db == nil || db.Dialector == nil {
		return DBDialectUnknown
	}
	return detectDBDialectName(db.Dialector.Name())
}

// UnixMilliTimeBucketExpr returns the SQL expressions for grouping a Unix-millisecond timestamp column by time bucket.
// The returned selectExpr is intended for Select, for example:
//
//	Select(fmt.Sprintf("%s as date, COUNT(*) as count", selectExpr))
//
// The returned groupExpr is intended for Group, for example:
//
//	Group(groupExpr)
//
// Workspace apps default to SQLite, so unknown dialects fall back to SQLite-compatible expressions.
func UnixMilliTimeBucketExpr(db *gorm.DB, column string, bucket TimeBucket) (selectExpr, groupExpr string) {
	return UnixMilliTimeBucketExprForDialect(DetectDBDialect(db), column, bucket)
}

// UnixMilliTimeBucketExprForDialect is the dialect-specific version of UnixMilliTimeBucketExpr.
// It returns a pair of expressions:
//   - selectExpr: use in Select(... as alias)
//   - groupExpr: use in Group(...)
func UnixMilliTimeBucketExprForDialect(dialect DBDialect, column string, bucket TimeBucket) (selectExpr, groupExpr string) {
	column = strings.TrimSpace(column)
	if column == "" {
		column = "created_at"
	}

	var expr string
	switch normalizeTimeBucket(bucket) {
	case TimeBucketHour:
		switch dialect {
		case DBDialectMySQL:
			expr = "DATE_FORMAT(FROM_UNIXTIME(" + column + "/1000), '%Y-%m-%d %H:00:00')"
		default:
			expr = "strftime('%Y-%m-%d %H:00:00', " + column + "/1000, 'unixepoch')"
		}
	case TimeBucketMonth:
		switch dialect {
		case DBDialectMySQL:
			expr = "DATE_FORMAT(FROM_UNIXTIME(" + column + "/1000), '%Y-%m')"
		default:
			expr = "strftime('%Y-%m', " + column + "/1000, 'unixepoch')"
		}
	default:
		switch dialect {
		case DBDialectMySQL:
			expr = "DATE_FORMAT(FROM_UNIXTIME(" + column + "/1000), '%Y-%m-%d')"
		default:
			expr = "strftime('%Y-%m-%d', " + column + "/1000, 'unixepoch')"
		}
	}

	return expr, expr
}

func detectDBDialectName(name string) DBDialect {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case string(DBDialectSQLite):
		return DBDialectSQLite
	case string(DBDialectMySQL):
		return DBDialectMySQL
	default:
		return DBDialectUnknown
	}
}

func normalizeTimeBucket(bucket TimeBucket) TimeBucket {
	switch TimeBucket(strings.ToLower(strings.TrimSpace(string(bucket)))) {
	case TimeBucketHour:
		return TimeBucketHour
	case TimeBucketMonth:
		return TimeBucketMonth
	default:
		return TimeBucketDay
	}
}
