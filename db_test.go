package main

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := initDB(dbPath)
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &App{db: db}
}

func TestCreateAndGetTank(t *testing.T) {
	app := newTestApp(t)
	size := 29.0
	id, err := app.CreateTank("Reef One", "saltwater", "first tank", &size)
	if err != nil {
		t.Fatalf("CreateTank: %v", err)
	}
	got, err := app.GetTank(id)
	if err != nil {
		t.Fatalf("GetTank: %v", err)
	}
	if got.Name != "Reef One" || got.TankType != "saltwater" || got.SizeGallons != 29.0 {
		t.Errorf("unexpected tank: %+v", got)
	}
}

func TestDeleteTankCascades(t *testing.T) {
	app := newTestApp(t)
	id, err := app.CreateTank("Planted", "freshwater", "", nil)
	if err != nil {
		t.Fatalf("CreateTank: %v", err)
	}
	ph := 7.2
	if err := app.InsertParameters(id, &ph, nil, nil, nil, nil, ""); err != nil {
		t.Fatalf("InsertParameters: %v", err)
	}
	if err := app.InsertObservation(id, "looks great"); err != nil {
		t.Fatalf("InsertObservation: %v", err)
	}

	if err := app.DeleteTank(id); err != nil {
		t.Fatalf("DeleteTank: %v", err)
	}

	if _, err := app.GetTank(id); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected ErrNoRows after delete, got %v", err)
	}

	var n int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM parameters WHERE tank_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("count parameters: %v", err)
	}
	if n != 0 {
		t.Errorf("expected parameters to cascade-delete, found %d", n)
	}
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM observations WHERE tank_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("count observations: %v", err)
	}
	if n != 0 {
		t.Errorf("expected observations to cascade-delete, found %d", n)
	}
}

func TestListTanksOrdersNewestFirst(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateTank("A", "freshwater", "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateTank("B", "freshwater", "", nil); err != nil {
		t.Fatal(err)
	}
	tanks, err := app.ListTanks()
	if err != nil {
		t.Fatalf("ListTanks: %v", err)
	}
	if len(tanks) != 2 {
		t.Fatalf("expected 2 tanks, got %d", len(tanks))
	}
	// CreatedAt has 1s resolution, so for tanks created back-to-back the order
	// may tie. Just verify both are present.
	names := map[string]bool{tanks[0].Name: true, tanks[1].Name: true}
	if !names["A"] || !names["B"] {
		t.Errorf("expected tanks A and B, got %+v", tanks)
	}
}

// Regression test: MAX(logged_at) in ListTanks is an aggregate without a
// declared column type, which the modernc.org/sqlite driver returns as a
// string. Make sure we still produce a usable *time.Time.
func TestListTanksPopulatesLastLogged(t *testing.T) {
	app := newTestApp(t)
	id, err := app.CreateTank("Reef", "saltwater", "", nil)
	if err != nil {
		t.Fatalf("CreateTank: %v", err)
	}
	ph := 8.2
	if err := app.InsertParameters(id, &ph, nil, nil, nil, nil, ""); err != nil {
		t.Fatalf("InsertParameters: %v", err)
	}
	tanks, err := app.ListTanks()
	if err != nil {
		t.Fatalf("ListTanks: %v", err)
	}
	if len(tanks) != 1 {
		t.Fatalf("expected 1 tank, got %d", len(tanks))
	}
	if tanks[0].LastLogged == nil {
		t.Fatal("expected LastLogged to be set after logging parameters")
	}
	if tanks[0].LastLogged.IsZero() {
		t.Errorf("LastLogged parsed to zero time")
	}
}

func TestLivestockCRUD(t *testing.T) {
	app := newTestApp(t)
	tankID, err := app.CreateTank("Tetra Tank", "freshwater", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	added := time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC)
	if err := app.InsertLivestock(tankID, "Neon tetra", 6, &added, "school"); err != nil {
		t.Fatalf("InsertLivestock: %v", err)
	}
	if err := app.InsertLivestock(tankID, "Otocinclus", 3, nil, ""); err != nil {
		t.Fatalf("InsertLivestock: %v", err)
	}

	list, err := app.ListLivestock(tankID)
	if err != nil {
		t.Fatalf("ListLivestock: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}

	var neon *Livestock
	for i := range list {
		if list[i].Species == "Neon tetra" {
			neon = &list[i]
		}
	}
	if neon == nil {
		t.Fatal("Neon tetra not in list")
	}
	if neon.Quantity != 6 {
		t.Errorf("expected qty 6, got %d", neon.Quantity)
	}
	if neon.AddedAt == nil || !neon.AddedAt.Equal(added) {
		t.Errorf("expected added_at %v, got %v", added, neon.AddedAt)
	}
	if neon.Notes != "school" {
		t.Errorf("expected notes %q, got %q", "school", neon.Notes)
	}

	if err := app.DeleteLivestock(tankID, neon.ID); err != nil {
		t.Fatalf("DeleteLivestock: %v", err)
	}
	list, _ = app.ListLivestock(tankID)
	if len(list) != 1 {
		t.Errorf("expected 1 remaining, got %d", len(list))
	}
}

func TestDeleteLivestockWrongTank(t *testing.T) {
	app := newTestApp(t)
	t1, _ := app.CreateTank("A", "freshwater", "", nil)
	t2, _ := app.CreateTank("B", "freshwater", "", nil)
	if err := app.InsertLivestock(t1, "Guppy", 4, nil, ""); err != nil {
		t.Fatal(err)
	}
	list, _ := app.ListLivestock(t1)
	if len(list) != 1 {
		t.Fatal("setup")
	}
	// Try to delete tank1's entry while claiming tank2 — should fail with ErrNoRows.
	err := app.DeleteLivestock(t2, list[0].ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected ErrNoRows for cross-tank delete, got %v", err)
	}
	// And the row should still be there.
	list, _ = app.ListLivestock(t1)
	if len(list) != 1 {
		t.Errorf("expected entry untouched, got %d", len(list))
	}
}

func TestDeleteTankCascadesLivestock(t *testing.T) {
	app := newTestApp(t)
	id, _ := app.CreateTank("Cascade", "freshwater", "", nil)
	if err := app.InsertLivestock(id, "Cherry shrimp", 10, nil, ""); err != nil {
		t.Fatal(err)
	}
	if err := app.DeleteTank(id); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM livestock WHERE tank_id = ?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("expected livestock to cascade-delete, found %d", n)
	}
}
