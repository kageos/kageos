package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureUserOrganizationMembership(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user-organization-membership?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatal(err)
	}
	for _, user := range []*User{
		{Username: "system", Email: "system@example.com"},
		{Username: "alice", Email: "alice@example.com"},
		{Username: "bob", Email: "bob@example.com", DepartmentFullPath: "/org/sales"},
	} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := ensureUserOrganizationMembership(db); err != nil {
		t.Fatal(err)
	}

	var system, alice, bob User
	if err := db.Where("username = ?", "system").First(&system).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("username = ?", "alice").First(&alice).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("username = ?", "bob").First(&bob).Error; err != nil {
		t.Fatal(err)
	}
	if system.DepartmentFullPath != "" {
		t.Fatalf("system department = %q, want empty", system.DepartmentFullPath)
	}
	if alice.DepartmentFullPath != "/org/unassigned" {
		t.Fatalf("alice department = %q, want /org/unassigned", alice.DepartmentFullPath)
	}
	if bob.DepartmentFullPath != "/org/sales" {
		t.Fatalf("bob department = %q, want /org/sales", bob.DepartmentFullPath)
	}
}
