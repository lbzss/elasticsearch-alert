package alert

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type Field struct {
	Key   string `json:"key" mapstructure:"key"`
	Count int    `json:"doc_count" mapstructure:"doc_count"`
}

type Record struct {
	Filter    string   `json:"filter,omitempty"`
	Text      string   `json:"text,omitempty"`
	BodyField bool     `json:"-"`
	Fields    []*Field `json:"fields,omitempty"`
}

type Alert struct {
	ID       string
	RuleName string
	Methods  []Method
	Records  []*Record
}

type Method interface {
	Write(context.Context, string, []*Record) error
}

type Handler struct {
	rand   *rand.Rand
	StopCh chan struct{}
	DoneCh chan struct{}
}

func NewHandler() *Handler {
	return &Handler{
		rand:   rand.New(rand.NewSource(int64(time.Now().Nanosecond()))),
		StopCh: make(chan struct{}),
		DoneCh: make(chan struct{}),
	}
}

func (h *Handler) Run(ctx context.Context, outputChan <-chan *Alert) {
	defer func() {
		close(h.DoneCh)
	}()

	alertCh := make(chan func() (int, error), 8)
	active := newInventory()

	alertFunc := func(ctx context.Context, alertID, rule string, method Method, records []*Record) func() (int, error) {
		return func() (int, error) {
			if active.remaining(alertID) < 1 {
				active.deregister(alertID)
				return 0, nil
			}
			active.decrement(alertID)
			err := method.Write(ctx, rule, records)
			return active.remaining(alertID), err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.StopCh:
			return
		case alert := <-outputChan:
			for i, method := range alert.Methods {
				alertMethodID := fmt.Sprintf("%d|%s", i, alert.ID)
				active.register(alertMethodID)
				alertCh <- alertFunc(ctx, alertMethodID, alert.RuleName, method, alert.Records)
			}
		case writeAlert := <-alertCh:
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, err := writeAlert()
			if err != nil {
				backoff := h.newBackoff()
				fmt.Println("error returned by alert function", "error", err, "remaining_retries", n, "backoff", backoff.String())
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
					alertCh <- writeAlert
				}
			}
		}
	}
}

func (h *Handler) newBackoff() time.Duration {
	return 2*time.Second + time.Duration(h.rand.Int63()%int64(time.Second*2)-int64(time.Second))
}
