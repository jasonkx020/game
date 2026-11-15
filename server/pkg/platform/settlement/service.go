package settlement

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/example/game/internal/game/engine"
	"github.com/example/game/pkg/platform/wallet"
)

const RakeRateNumerator = int64(5)
const RakeRateDenominator = int64(100)

// CoinDelta is the net game-coin change for one player after rake.
type CoinDelta struct {
	UserID  uint64
	Delta   int64
	AuditSN uint64
}

type Result struct {
	CoinDeltas []CoinDelta
	RakeAmount int64
}

type Service struct {
	wallet *wallet.Service
}

func New(w *wallet.Service) *Service {
	return &Service{wallet: w}
}

// SkipUser reports whether monetary settlement should skip this user (e.g. bots).
type SkipUser func(userID uint64) bool

// ApplyScoreField applies rule_score × baseScore to game_coin and takes 5% rake
// from the winner's net win. room_card mode should not call this.
func (s *Service) ApplyScoreField(
	ctx context.Context,
	tx *sqlx.Tx,
	gameID string,
	roomID, roundID uuid.UUID,
	baseScore int64,
	settle engine.SettlementResult,
	nextAuditSN func() uint64,
	skip SkipUser,
) (*Result, error) {
	if baseScore <= 0 {
		baseScore = 1
	}
	if !settle.Valid {
		return &Result{}, nil
	}

	type raw struct {
		userID uint64
		delta  int64
	}
	raws := make([]raw, 0, len(settle.Scores))
	var winnerNet int64
	for _, sc := range settle.Scores {
		if skip != nil && skip(sc.UserID) {
			continue
		}
		d := sc.RuleScore * baseScore
		raws = append(raws, raw{userID: sc.UserID, delta: d})
		if sc.UserID == settle.WinnerID && d > 0 {
			winnerNet = d
		}
	}
	if winnerNet == 0 {
		for _, r := range raws {
			if r.delta > winnerNet {
				winnerNet = r.delta
			}
		}
	}
	rake := winnerNet * RakeRateNumerator / RakeRateDenominator
	if rake < 0 {
		rake = 0
	}

	refID := roundID.String()
	out := &Result{RakeAmount: rake, CoinDeltas: make([]CoinDelta, 0, len(raws))}

	for _, r := range raws {
		delta := r.delta
		if settle.WinnerID != 0 && r.userID == settle.WinnerID {
			delta -= rake
		}
		sn := nextAuditSN()
		reason := "settlement"
		applied, err := s.wallet.ApplyGameCoinDeltaTx(ctx, tx, int64(r.userID), gameID, delta, reason, refID, sn)
		if err == wallet.ErrInsufficientBalance && delta < 0 {
			// Clamp loss to emptying the wallet (bankrupt).
			applied, delta, sn, err = s.clampLoss(ctx, tx, int64(r.userID), gameID, reason, refID, nextAuditSN)
		}
		if err != nil {
			return nil, fmt.Errorf("settle user %d: %w", r.userID, err)
		}
		_ = applied
		out.CoinDeltas = append(out.CoinDeltas, CoinDelta{UserID: r.userID, Delta: delta, AuditSN: sn})
	}

	_ = roomID
	return out, nil
}

func (s *Service) clampLoss(
	ctx context.Context,
	tx *sqlx.Tx,
	userID int64,
	gameID, reason, refID string,
	nextAuditSN func() uint64,
) (newBal int64, delta int64, sn uint64, err error) {
	if _, err = tx.ExecContext(ctx,
		`INSERT INTO wallet_game_coin (user_id, game_id, balance) VALUES ($1,$2,0)
		 ON CONFLICT (user_id, game_id) DO NOTHING`, userID, gameID); err != nil {
		return 0, 0, 0, err
	}
	var bal int64
	if err = tx.GetContext(ctx, &bal,
		`SELECT balance FROM wallet_game_coin WHERE user_id=$1 AND game_id=$2 FOR UPDATE`, userID, gameID); err != nil {
		return 0, 0, 0, err
	}
	if bal == 0 {
		return 0, 0, 0, nil
	}
	sn = nextAuditSN()
	newBal, err = s.wallet.ApplyGameCoinDeltaTx(ctx, tx, userID, gameID, -bal, reason, refID, sn)
	return newBal, -bal, sn, err
}

// ComputeOnly calculates coin deltas without writing the wallet (tests / room_card no-op path).
func ComputeOnly(baseScore int64, settle engine.SettlementResult, skip SkipUser) *Result {
	if baseScore <= 0 {
		baseScore = 1
	}
	raws := make([]CoinDelta, 0, len(settle.Scores))
	var winnerNet int64
	for _, sc := range settle.Scores {
		if skip != nil && skip(sc.UserID) {
			continue
		}
		d := sc.RuleScore * baseScore
		raws = append(raws, CoinDelta{UserID: sc.UserID, Delta: d})
		if sc.UserID == settle.WinnerID && d > 0 {
			winnerNet = d
		}
	}
	rake := winnerNet * RakeRateNumerator / RakeRateDenominator
	if rake < 0 {
		rake = 0
	}
	out := &Result{RakeAmount: rake, CoinDeltas: make([]CoinDelta, 0, len(raws))}
	for _, r := range raws {
		delta := r.Delta
		if settle.WinnerID != 0 && r.UserID == settle.WinnerID {
			delta -= rake
		}
		out.CoinDeltas = append(out.CoinDeltas, CoinDelta{UserID: r.UserID, Delta: delta})
	}
	return out
}
