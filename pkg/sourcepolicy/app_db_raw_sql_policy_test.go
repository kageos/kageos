package sourcepolicy

import "testing"

func TestAppDBRawSQLPolicyAllowsReadOnlyQueries(t *testing.T) {
	t.Parallel()

	policy := newReadOnlyAppDBRawSQLPolicy()
	for _, sqlText := range []string{
		"SELECT id, name FROM tickets WHERE deleted_at IS NULL",
		"WITH stats AS (SELECT status, COUNT(*) c FROM tickets GROUP BY status) SELECT * FROM stats",
		"-- report\nSELECT 'update' AS word, `delete` AS quoted_identifier FROM tickets",
		"SELECT JSON_SET(payload, '$.status', 'update') AS payload FROM tickets",
		"SELECT * FROM tickets /* update tickets set status = 'done' */ WHERE id = ?",
	} {
		sqlText := sqlText
		t.Run(sqlText, func(t *testing.T) {
			t.Parallel()
			if issue := policy.ValidateSQL(sqlText); issue != "" {
				t.Fatalf("ValidateSQL(%q) issue = %s", sqlText, issue)
			}
		})
	}
}

func TestAppDBRawSQLPolicyRejectsNonReadOnlyQueries(t *testing.T) {
	t.Parallel()

	policy := newReadOnlyAppDBRawSQLPolicy()
	for _, sqlText := range []string{
		"",
		"UPDATE tickets SET status = 'done'",
		"INSERT INTO tickets(title) VALUES ('x')",
		"DELETE FROM tickets WHERE id = 1",
		"CREATE TEMPORARY TABLE tmp_stats (id bigint)",
		"SET sql_mode = ''",
		"SELECT * FROM tickets; UPDATE tickets SET status = 'done'",
		"WITH changed AS (UPDATE tickets SET status = 'done') SELECT * FROM changed",
	} {
		sqlText := sqlText
		t.Run(sqlText, func(t *testing.T) {
			t.Parallel()
			if issue := policy.ValidateSQL(sqlText); issue == "" {
				t.Fatalf("ValidateSQL(%q) should reject", sqlText)
			}
		})
	}
}
