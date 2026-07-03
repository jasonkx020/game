package metrics

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type Overview struct {
	GMVTodayCents      int64 `json:"gmv_today_cents"`
	RoomCardsSoldToday int64 `json:"room_cards_sold_today"`
	RoomCardsUsedToday int64 `json:"room_cards_used_today"`
	ActiveClubs        int64 `json:"active_clubs"`
	RoomsCreatedToday  int64 `json:"rooms_created_today"`
}

type RoomCardDay struct {
	Date string `json:"date"`
	Sold int64  `json:"sold"`
	Used int64  `json:"used"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	start := startOfDay(time.Now())
	var o Overview

	_ = s.db.GetContext(ctx, &o.GMVTodayCents,
		`SELECT COALESCE(SUM(amount_cny * 100), 0) FROM recharge_order WHERE created_at >= $1`, start)
	_ = s.db.GetContext(ctx, &o.RoomCardsSoldToday,
		`SELECT COALESCE(SUM(cards), 0) FROM recharge_order WHERE created_at >= $1`, start)
	_ = s.db.GetContext(ctx, &o.RoomCardsUsedToday,
		`SELECT COALESCE(SUM(ABS(delta)), 0) FROM wallet_ledger
		 WHERE wallet_type='room_card' AND delta < 0 AND reason='create_room' AND created_at >= $1`, start)
	_ = s.db.GetContext(ctx, &o.ActiveClubs,
		`SELECT COUNT(*) FROM club WHERE status='active'`)
	_ = s.db.GetContext(ctx, &o.RoomsCreatedToday,
		`SELECT COUNT(*) FROM room WHERE created_at >= $1`, start)

	return &o, nil
}

func (s *Service) RoomCardTrend(ctx context.Context, days int) ([]RoomCardDay, error) {
	if days <= 0 {
		days = 7
	}
	start := startOfDay(time.Now().AddDate(0, 0, -(days - 1)))

	type row struct {
		Day  time.Time `db:"day"`
		Sold int64     `db:"sold"`
		Used int64     `db:"used"`
	}
	var rows []row

	err := s.db.SelectContext(ctx, &rows, `
WITH days AS (
  SELECT generate_series($1::date, CURRENT_DATE, '1 day'::interval)::date AS day
)
SELECT d.day,
  COALESCE((SELECT SUM(cards) FROM recharge_order r WHERE r.created_at::date = d.day), 0) AS sold,
  COALESCE((SELECT SUM(ABS(delta)) FROM wallet_ledger w
    WHERE w.wallet_type='room_card' AND w.delta < 0 AND w.reason='create_room'
    AND w.created_at::date = d.day), 0) AS used
FROM days d ORDER BY d.day`, start)
	if err != nil {
		return nil, err
	}

	out := make([]RoomCardDay, len(rows))
	for i, r := range rows {
		out[i] = RoomCardDay{
			Date: r.Day.Format("2006-01-02"),
			Sold: r.Sold,
			Used: r.Used,
		}
	}
	return out, nil
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
