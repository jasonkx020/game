package kyc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalidID   = errors.New("invalid id card number")
	ErrInvalidName = errors.New("invalid real name")
	ErrMismatch    = errors.New("identity mismatch")
)

type Result struct {
	Matched    bool
	BirthDate  *time.Time
	IsMinor    bool
	IDHash     string
	IDMasked   string
	NameMasked string
}

type Verifier interface {
	Verify(ctx context.Context, realName, idCardNo string) (*Result, error)
}

type MockVerifier struct{}

func NewMock() *MockVerifier { return &MockVerifier{} }

func (m *MockVerifier) Verify(ctx context.Context, realName, idCardNo string) (*Result, error) {
	_ = ctx
	name := strings.TrimSpace(realName)
	id := strings.ToUpper(strings.TrimSpace(idCardNo))
	if utf8.RuneCountInString(name) < 2 {
		return nil, ErrInvalidName
	}
	if !validIDCard(id) {
		return nil, ErrInvalidID
	}
	// Dev rule: real name "拒绝" forces mismatch for negative tests.
	matched := name != "拒绝"
	birth, isMinor := parseBirthMinor(id)
	res := &Result{
		Matched:    matched,
		BirthDate:  birth,
		IsMinor:    isMinor,
		IDHash:     HashID(id),
		IDMasked:   MaskID(id),
		NameMasked: MaskName(name),
	}
	if !matched {
		return res, ErrMismatch
	}
	return res, nil
}

// AliyunVerifier calls Id2MetaVerify. Uses HTTP-style interface abstraction so
// unit tests can inject mocks without the SDK; live wiring is in aliyun.go.
type AliyunClient interface {
	Id2MetaVerify(ctx context.Context, name, idNo string) (bizCode string, err error)
}

type AliyunVerifier struct {
	Client AliyunClient
}

func (a *AliyunVerifier) Verify(ctx context.Context, realName, idCardNo string) (*Result, error) {
	name := strings.TrimSpace(realName)
	id := strings.ToUpper(strings.TrimSpace(idCardNo))
	if utf8.RuneCountInString(name) < 2 {
		return nil, ErrInvalidName
	}
	if !validIDCard(id) {
		return nil, ErrInvalidID
	}
	biz, err := a.Client.Id2MetaVerify(ctx, name, id)
	if err != nil {
		return nil, err
	}
	birth, isMinor := parseBirthMinor(id)
	res := &Result{
		BirthDate:  birth,
		IsMinor:    isMinor,
		IDHash:     HashID(id),
		IDMasked:   MaskID(id),
		NameMasked: MaskName(name),
	}
	if biz == "1" {
		res.Matched = true
		return res, nil
	}
	res.Matched = false
	return res, ErrMismatch
}

func HashID(id string) string {
	sum := sha256.Sum256([]byte(strings.ToUpper(id)))
	return hex.EncodeToString(sum[:])
}

func MaskID(id string) string {
	if len(id) < 8 {
		return "****"
	}
	return id[:4] + "**********" + id[len(id)-4:]
}

func MaskName(name string) string {
	r := []rune(strings.TrimSpace(name))
	if len(r) == 0 {
		return "*"
	}
	if len(r) == 1 {
		return string(r[0])
	}
	return string(r[0]) + strings.Repeat("*", len(r)-1)
}

var idRe = regexp.MustCompile(`^\d{17}[\dX]$`)

func validIDCard(id string) bool {
	if !idRe.MatchString(id) {
		return false
	}
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checks := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}
	sum := 0
	for i := 0; i < 17; i++ {
		n, _ := strconv.Atoi(string(id[i]))
		sum += n * weights[i]
	}
	return checks[sum%11] == id[17]
}

func parseBirthMinor(id string) (*time.Time, bool) {
	if len(id) < 14 {
		return nil, false
	}
	y, _ := strconv.Atoi(id[6:10])
	m, _ := strconv.Atoi(id[10:12])
	d, _ := strconv.Atoi(id[12:14])
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	age := ageAt(t, time.Now())
	return &t, age < 18
}

func ageAt(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.YearDay() < birth.YearDay() {
		age--
	}
	return age
}

func FormatErr(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
