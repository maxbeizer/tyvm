package main

import "time"

type Tank struct {
	ID          int64
	Name        string
	SizeGallons float64
	TankType    string
	Notes       string
	CreatedAt   time.Time
}

type TankWithLastLog struct {
	Tank
	LastLogged *time.Time
}

type Parameter struct {
	ID       int64
	TankID   int64
	PH       *float64
	Ammonia  *float64
	Nitrite  *float64
	Nitrate  *float64
	TempF    *float64
	Notes    string
	LoggedAt time.Time
}

type Observation struct {
	ID         int64
	TankID     int64
	Note       string
	ObservedAt time.Time
}

type Livestock struct {
	ID        int64
	TankID    int64
	Species   string
	Quantity  int
	AddedAt   *time.Time
	Notes     string
	CreatedAt time.Time
}
