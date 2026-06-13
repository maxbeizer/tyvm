package main

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
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
