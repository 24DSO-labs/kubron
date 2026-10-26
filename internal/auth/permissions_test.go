// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import "testing"

func TestAdminHasEverything(t *testing.T) {
	for _, p := range allPermissions {
		if !RoleAdmin.Has(p) {
			t.Errorf("admin должен иметь %q", p)
		}
	}
	// в том числе права, которых пока нет
	if !RoleAdmin.Has(Permission("future.permission")) {
		t.Error("admin должен проходить любую проверку, в том числе для будущих прав")
	}
}

func TestUserIsReadOnly(t *testing.T) {
	allowed := map[Permission]bool{PermRead: true, PermSecretsRead: true}
	for _, p := range allPermissions {
		if got := RoleUser.Has(p); got != allowed[p] {
			t.Errorf("user.Has(%q) = %v, ожидалось %v", p, got, allowed[p])
		}
	}
}

func TestOperatorCanOperateButNotDestroy(t *testing.T) {
	allowed := map[Permission]bool{PermRead: true, PermSecretsRead: true, PermOperate: true}
	for _, p := range allPermissions {
		if got := RoleOperator.Has(p); got != allowed[p] {
			t.Errorf("operator.Has(%q) = %v, ожидалось %v", p, got, allowed[p])
		}
	}
	// оператор не слабее пользователя: всё, что видит user, видит и operator
	for _, p := range RoleUser.Permissions() {
		if !RoleOperator.Has(p) {
			t.Errorf("оператор лишён права %q, которое есть у пользователя", p)
		}
	}
}

func TestUnknownRoleHasNothing(t *testing.T) {
	r := Role("hacker")
	if r.Valid() {
		t.Fatal("неизвестная роль не должна быть валидной")
	}
	for _, p := range allPermissions {
		if r.Has(p) {
			t.Errorf("неизвестная роль получила право %q", p)
		}
	}
	if len(r.Permissions()) != 0 {
		t.Error("у неизвестной роли не должно быть прав")
	}
}

func TestRolesListMatchesTable(t *testing.T) {
	list := Roles()
	if len(list) != len(roles) {
		t.Fatalf("Roles() вернула %d ролей, в таблице %d: забыли добавить роль в roleOrder?", len(list), len(roles))
	}
	for _, ri := range list {
		if !ri.ID.Valid() || ri.Label == "" {
			t.Errorf("некорректное описание роли: %+v", ri)
		}
	}
}

func TestRolePermissionsAreKnown(t *testing.T) {
	for id, d := range roles {
		for _, p := range d.permissions {
			if !KnownPermission(p) {
				t.Errorf("роль %s содержит неизвестное право %q", id, p)
			}
		}
	}
}
