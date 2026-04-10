package app

import "testing"

func TestUnixMilliTimeBucketExprForDialect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dialect    DBDialect
		column     string
		bucket     TimeBucket
		wantSelect string
	}{
		{
			name:       "sqlite day",
			dialect:    DBDialectSQLite,
			column:     "created_at",
			bucket:     TimeBucketDay,
			wantSelect: "strftime('%Y-%m-%d', created_at/1000, 'unixepoch')",
		},
		{
			name:       "mysql hour",
			dialect:    DBDialectMySQL,
			column:     "paid_at",
			bucket:     TimeBucketHour,
			wantSelect: "DATE_FORMAT(FROM_UNIXTIME(paid_at/1000), '%Y-%m-%d %H:00:00')",
		},
		{
			name:       "mysql month",
			dialect:    DBDialectMySQL,
			column:     "paid_at",
			bucket:     TimeBucketMonth,
			wantSelect: "DATE_FORMAT(FROM_UNIXTIME(paid_at/1000), '%Y-%m')",
		},
		{
			name:       "unknown dialect defaults to sqlite",
			dialect:    DBDialectUnknown,
			column:     "created_at",
			bucket:     TimeBucketDay,
			wantSelect: "strftime('%Y-%m-%d', created_at/1000, 'unixepoch')",
		},
		{
			name:       "blank column defaults to created_at",
			dialect:    DBDialectSQLite,
			column:     "",
			bucket:     TimeBucketMonth,
			wantSelect: "strftime('%Y-%m', created_at/1000, 'unixepoch')",
		},
		{
			name:       "unknown bucket defaults to day",
			dialect:    DBDialectSQLite,
			column:     "created_at",
			bucket:     TimeBucket("year"),
			wantSelect: "strftime('%Y-%m-%d', created_at/1000, 'unixepoch')",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selectExpr, groupExpr := UnixMilliTimeBucketExprForDialect(tt.dialect, tt.column, tt.bucket)
			if selectExpr != tt.wantSelect {
				t.Fatalf("select expr mismatch\nwant: %s\ngot:  %s", tt.wantSelect, selectExpr)
			}
			if groupExpr != tt.wantSelect {
				t.Fatalf("group expr mismatch\nwant: %s\ngot:  %s", tt.wantSelect, groupExpr)
			}
		})
	}
}

func TestDetectDBDialectName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want DBDialect
	}{
		{name: "sqlite", in: "sqlite", want: DBDialectSQLite},
		{name: "mysql uppercase", in: " MySQL ", want: DBDialectMySQL},
		{name: "unknown", in: "postgres", want: DBDialectUnknown},
		{name: "blank", in: "", want: DBDialectUnknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := detectDBDialectName(tt.in); got != tt.want {
				t.Fatalf("want %s, got %s", tt.want, got)
			}
		})
	}
}
