package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRemoveRetiredCompanySchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:retired-company-schema?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"CREATE TABLE `user` (`id` INTEGER PRIMARY KEY, `company_code` TEXT)",
		"CREATE TABLE `auth_oauth_registration_intents` (`id` INTEGER PRIMARY KEY, `company_code` TEXT)",
		"CREATE TABLE `company` (`id` INTEGER PRIMARY KEY, `code` TEXT)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := removeRetiredCompanySchema(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn(&User{}, "company_code") {
		t.Fatal("user.company_code should be removed")
	}
	if db.Migrator().HasColumn(&AuthOAuthRegistrationIntent{}, "company_code") {
		t.Fatal("auth_oauth_registration_intents.company_code should be removed")
	}
	if db.Migrator().HasTable("company") {
		t.Fatal("company table should be removed")
	}
}
