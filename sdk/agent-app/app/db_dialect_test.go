package app

import "testing"

func TestDetectDBDialectName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want DBDialect
	}{
		{name: "mysql uppercase", in: " MySQL ", want: DBDialectMySQL},
		{name: "sqlite is not an app business dialect", in: "sqlite", want: DBDialectUnknown},
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

func TestDateTimeBucketExprForDialect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dialect    DBDialect
		column     string
		bucket     TimeBucket
		wantSelect string
	}{
		{
			name:       "mysql day",
			dialect:    DBDialectMySQL,
			column:     "created_at",
			bucket:     TimeBucketDay,
			wantSelect: "DATE_FORMAT(created_at, '%Y-%m-%d')",
		},
		{
			name:       "mysql hour",
			dialect:    DBDialectMySQL,
			column:     "paid_at",
			bucket:     TimeBucketHour,
			wantSelect: "DATE_FORMAT(paid_at, '%Y-%m-%d %H:00:00')",
		},
		{
			name:       "blank column defaults to created_at",
			dialect:    DBDialectUnknown,
			column:     "",
			bucket:     TimeBucketMonth,
			wantSelect: "DATE_FORMAT(created_at, '%Y-%m')",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selectExpr, groupExpr := DateTimeBucketExprForDialect(tt.dialect, tt.column, tt.bucket)
			if selectExpr != tt.wantSelect {
				t.Fatalf("select expr mismatch\nwant: %s\ngot:  %s", tt.wantSelect, selectExpr)
			}
			if groupExpr != tt.wantSelect {
				t.Fatalf("group expr mismatch\nwant: %s\ngot:  %s", tt.wantSelect, groupExpr)
			}
		})
	}
}
