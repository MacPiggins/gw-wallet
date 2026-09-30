package broker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/MacPiggins/gw-wallet/internal/config"
	"github.com/MacPiggins/gw-wallet/internal/model"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Nats struct {
	conn *nats.Conn
	js   jetstream.JetStream
}

func NewNats(ctx context.Context, conf *config.Config) (*Nats, error) {
	nc, err := nats.Connect(conf.NatsConnstr)
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect to nats", slog.Any("error", err))
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect to nats jetstream", slog.Any("error", err))
		nc.Close()
		return nil, err
	}

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        conf.NatsStream,
		Description: "Wallet transfers and events",
		Subjects:    []string{"wallet.transfers", "wallet.events"},
		Storage:     jetstream.FileStorage,
		Retention:   jetstream.LimitsPolicy,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to init js stream", slog.Any("error", err))
		nc.Close()
		return nil, err
	}

	return &Nats{conn: nc, js: js}, nil
}

// Close closes the NATS connection.
func (n *Nats) Close() {
	n.conn.Close()
}

func (n *Nats) SendTransfer(ctx context.Context, message *model.Transfer) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = n.js.PublishMsg(ctx, &nats.Msg{
		Subject: "wallet.transfers",
		Data:    data,
	})
	return err
}

func (n *Nats) SendEvent(ctx context.Context, message *model.Event) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = n.js.PublishMsg(ctx, &nats.Msg{
		Subject: "wallet.events",
		Data:    data,
	})
	return err
}
