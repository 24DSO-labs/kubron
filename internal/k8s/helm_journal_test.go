// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

func jrec(id, op, status, rel string) HelmOperationRecord {
	return HelmOperationRecord{
		ID: id, Op: op, Status: status, Release: rel, Namespace: "prod", Cluster: "dev", Detail: "d-" + rel,
		StartedAt: "2026-10-28T10:00:00Z", FinishedAt: "2026-10-28T10:00:05Z", DurationMs: 5000, Log: []string{"line1", "line2"},
	}
}

func TestJournalAddListGet(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()
	journalAdd(jrec("op-aaaaaaa1", "install", "success", "web"))
	journalAdd(jrec("op-aaaaaaa2", "rollback", "failed", "api"))
	journalAdd(jrec("op-aaaaaaa3", "upgrade", "canceled", "web"))

	all, err := m.ListHelmOperations(HelmOperationQuery{})
	if err != nil || len(all) != 3 || all[0].ID != "op-aaaaaaa3" || all[2].ID != "op-aaaaaaa1" {
		t.Fatalf("новые должны идти первыми: %+v %v", all, err)
	}
	for _, r := range all {
		if r.Log != nil {
			t.Error("в списке журнал Helm не отдаётся")
		}
	}
	if runtime.GOOS != "windows" {
		if st, err := os.Stat(helmJournalPath()); err != nil || st.Mode().Perm() != 0o600 {
			t.Errorf("права operations.json: %v %v", st, err)
		}
	}

	got, err := m.GetHelmOperation("op-aaaaaaa2")
	if err != nil || got.Release != "api" || len(got.Log) != 2 {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := m.GetHelmOperation("op-zzzzzzz9"); err == nil {
		t.Error("несуществующая запись")
	}
	if _, err := m.GetHelmOperation("../../etc"); err == nil {
		t.Error("некорректный идентификатор")
	}
}

func TestJournalFilters(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()
	journalAdd(jrec("op-bbbbbbb1", "install", "success", "web"))
	journalAdd(jrec("op-bbbbbbb2", "rollback", "failed", "api"))
	journalAdd(jrec("op-bbbbbbb3", "install", "failed", "web-2"))

	count := func(q HelmOperationQuery) int {
		got, err := m.ListHelmOperations(q)
		if err != nil {
			t.Fatal(err)
		}
		return len(got)
	}
	if n := count(HelmOperationQuery{Op: "install"}); n != 2 {
		t.Errorf("op=install: %d", n)
	}
	if n := count(HelmOperationQuery{Status: "failed"}); n != 2 {
		t.Errorf("status=failed: %d", n)
	}
	if n := count(HelmOperationQuery{Op: "install", Status: "failed"}); n != 1 {
		t.Errorf("комбинация: %d", n)
	}
	if n := count(HelmOperationQuery{Search: "WEB"}); n != 2 {
		t.Errorf("поиск без учёта регистра: %d", n)
	}
	if n := count(HelmOperationQuery{Search: "web 2"}); n != 1 { // все слова должны находиться в одной записи: web-2
		t.Errorf("поиск по нескольким словам: %d", n)
	}
	if n := count(HelmOperationQuery{Search: "web api"}); n != 0 {
		t.Errorf("слова из разных записей не совпадают: %d", n)
	}
	if n := count(HelmOperationQuery{Limit: 1}); n != 1 {
		t.Errorf("limit: %d", n)
	}
	if _, err := m.ListHelmOperations(HelmOperationQuery{Op: "drop"}); err == nil {
		t.Error("неизвестная операция")
	}
	if _, err := m.ListHelmOperations(HelmOperationQuery{Status: "weird"}); err == nil {
		t.Error("неизвестный статус")
	}
}

func TestJournalCapAndClear(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()
	for i := 0; i < helmJournalMax+5; i++ {
		journalAdd(jrec(fmt.Sprintf("op-%08d", i), "install", "success", "web"))
	}
	all, err := m.ListHelmOperations(HelmOperationQuery{Limit: 1000})
	if err != nil || len(all) != helmJournalMax {
		t.Fatalf("журнал должен хранить не больше %d записей: %d %v", helmJournalMax, len(all), err)
	}
	if all[0].ID != fmt.Sprintf("op-%08d", helmJournalMax+4) {
		t.Errorf("самая новая запись первой: %s", all[0].ID)
	}
	if err := m.ClearHelmOperations(); err != nil {
		t.Fatal(err)
	}
	if all, _ := m.ListHelmOperations(HelmOperationQuery{}); len(all) != 0 {
		t.Errorf("после очистки: %d", len(all))
	}
	if err := m.ClearHelmOperations(); err != nil {
		t.Errorf("повторная очистка безопасна: %v", err)
	}
}

func TestJournalRecoversFromCorruption(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()
	if err := os.MkdirAll(helmHomeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(helmJournalPath(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ListHelmOperations(HelmOperationQuery{}); err == nil {
		t.Error("повреждённый журнал должен давать понятную ошибку")
	}
	journalAdd(jrec("op-ccccccc1", "install", "success", "web")) // запись заменяет повреждённый файл
	if all, err := m.ListHelmOperations(HelmOperationQuery{}); err != nil || len(all) != 1 {
		t.Errorf("после записи журнал снова читается: %v %v", all, err)
	}
}

func TestTrackerFinishWritesJournal(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()

	run := func(id string, err error) HelmOperationRecord {
		_, tr, e := m.beginHelmOp(context.Background(), id, "upgrade", "prod", "web", true)
		if e != nil {
			t.Fatal(e)
		}
		tr.note("изменение values")
		tr.setRevision(7)
		tr.addLine("waiting for resources count=3")
		tr.finish(err)
		rec, e := m.GetHelmOperation(id)
		if e != nil {
			t.Fatal(e)
		}
		return *rec
	}

	ok := run("op-ddddddd1", nil)
	if ok.Status != "success" || ok.Error != "" || ok.Revision != 7 || ok.Detail != "изменение values" ||
		ok.Op != "upgrade" || ok.Release != "web" || len(ok.Log) != 1 || ok.StartedAt == "" || ok.FinishedAt == "" {
		t.Errorf("success: %+v", ok)
	}
	if bad := run("op-ddddddd2", errors.New("timed out waiting")); bad.Status != "failed" || !strings.Contains(bad.Error, "timed out") {
		t.Errorf("failed: %+v", bad)
	}
	if cn := run("op-ddddddd3", context.Canceled); cn.Status != "canceled" || cn.Error != "операция отменена" {
		t.Errorf("canceled: %+v", cn)
	}
	// каждая операция — ровно одна запись
	all, _ := m.ListHelmOperations(HelmOperationQuery{})
	if len(all) != 3 {
		t.Errorf("записей: %d", len(all))
	}
}

func TestTrackerLogRingKeepsLastLines(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()
	_, tr, _ := m.beginHelmOp(context.Background(), "op-eeeeeee1", "install", "prod", "web", false)
	for i := 0; i < helmJournalLogLines+20; i++ {
		tr.addLine(fmt.Sprintf("line %d", i))
	}
	tr.finish(nil)
	rec, err := m.GetHelmOperation("op-eeeeeee1")
	if err != nil || len(rec.Log) != helmJournalLogLines {
		t.Fatalf("%d строк: %v", len(rec.Log), err)
	}
	if rec.Log[len(rec.Log)-1] != fmt.Sprintf("line %d", helmJournalLogLines+19) || rec.Log[0] != "line 20" {
		t.Errorf("должны сохраниться последние строки: %q … %q", rec.Log[0], rec.Log[len(rec.Log)-1])
	}
}
