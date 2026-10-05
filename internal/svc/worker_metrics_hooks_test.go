package svc

import (
	"context"
	"database/sql"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	_ "modernc.org/sqlite"
	"testing"
)

func TestWorkerMetricsGlobalWhileDataTargetsStayTenantScoped(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if _, err = database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	db := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, database)))
	t.Cleanup(func() { _ = db.Close() })
	if err = db.Schema.Create(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err = setupIOHooks(db); err != nil {
		t.Fatal(err)
	}
	ctx1 := hooks.SetTenantIDToContext(context.Background(), 1)
	ctx2 := hooks.SetTenantIDToContext(context.Background(), 2)
	metric, err := db.WorkerMetrics.Create().SetWorkerID("demo-worker").SetWorkerName("demo").Save(ctx1)
	if err != nil {
		t.Fatalf("global metrics create must not set nonexistent tenant field: %v", err)
	}
	for _, ctx := range []context.Context{ctx1, ctx2} {
		rows, err := db.WorkerMetrics.Query().All(ctx)
		if err != nil || len(rows) != 1 || rows[0].ID != metric.ID {
			t.Fatalf("global metrics query injected nonexistent tenant column: %v", err)
		}
	}
	if _, err = db.WorkerMetrics.Query().All(context.Background()); err == nil {
		t.Fatal("metrics query must still require authenticated tenant context")
	}
	for i, ctx := range []context.Context{ctx1, ctx2} {
		_, err = db.DataTarget.Create().SetTargetName("demo").SetTargetCode([]string{"tenant-one", "tenant-two"}[i]).SetTargetType("file").Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i, ctx := range []context.Context{ctx1, ctx2} {
		rows, err := db.DataTarget.Query().All(ctx)
		if err != nil || len(rows) != 1 || rows[0].TenantID != uint64(i+1) {
			t.Fatalf("tenant data leaked: rows=%v err=%v", rows, err)
		}
	}
	if _, err = db.DataTarget.Query().All(context.Background()); err == nil {
		t.Fatal("tenant entity query without tenant must fail")
	}
	if _, err = db.DataTarget.Create().SetTargetName("missing").SetTargetCode("missing").SetTargetType("file").Save(context.Background()); err == nil {
		t.Fatal("tenant entity create without tenant must fail")
	}
}
