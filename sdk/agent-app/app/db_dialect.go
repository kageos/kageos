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

// DateTimeBucketExpr returns SQL expressions for grouping a native datetime column by time bucket.
func DateTimeBucketExpr(db *gorm.DB, column string, bucket TimeBucket) (selectExpr, groupExpr string) {
	return DateTimeBucketExprForDialect(DetectDBDialect(db), column, bucket)
}

// DateTimeBucketExprForDialect is the dialect-specific version of DateTimeBucketExpr.
// It expects the column to be a real datetime/time column, or SQLite text in "YYYY-MM-DD HH:mm:ss" format.
func DateTimeBucketExprForDialect(dialect DBDialect, column string, bucket TimeBucket) (selectExpr, groupExpr string) {
	column = strings.TrimSpace(column)
	if column == "" {
		column = "created_at"
	}

	var expr string
	switch normalizeTimeBucket(bucket) {
	case TimeBucketHour:
		switch dialect {
		case DBDialectMySQL:
			expr = "DATE_FORMAT(" + column + ", '%Y-%m-%d %H:00:00')"
		default:
			expr = "strftime('%Y-%m-%d %H:00:00', " + column + ")"
		}
	case TimeBucketMonth:
		switch dialect {
		case DBDialectMySQL:
			expr = "DATE_FORMAT(" + column + ", '%Y-%m')"
		default:
			expr = "strftime('%Y-%m', " + column + ")"
		}
	default:
		switch dialect {
		case DBDialectMySQL:
			expr = "DATE_FORMAT(" + column + ", '%Y-%m-%d')"
		default:
			expr = "strftime('%Y-%m-%d', " + column + ")"
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
