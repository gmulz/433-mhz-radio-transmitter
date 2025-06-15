package rftransmitter433mhz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/utils"
	"go.viam.com/utils/rpc"
)

var (
	RfTransmitter    = resource.NewModel("grant-dev", "rf-transmitter-433mhz", "rf-transmitter")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterComponent(generic.API, RfTransmitter,
		resource.Registration[resource.Resource, *Config]{
			Constructor: newRfTransmitter433MhzRfTransmitter,
		},
	)
}

type Config struct {
	Board   string `json:"board"`
	DataPin string `json:"data_pin"`
}

// Validate ensures all parts of the config are valid and important fields exist.
// Returns implicit required (first return) and optional (second return) dependencies based on the config.
// The path is the JSON path in your robot's config (not the `Config` struct) to the
// resource being validated; e.g. "components.0".
func (cfg *Config) Validate(path string) ([]string, []string, error) {
	if cfg.Board == "" {
		return nil, nil, utils.NewConfigValidationFieldRequiredError(path, "board")
	}
	if cfg.DataPin == "" {
		return nil, nil, utils.NewConfigValidationFieldRequiredError(path, "data_pin")
	}
	return []string{cfg.Board}, nil, nil
}

type rfTransmitter433MhzRfTransmitter struct {
	resource.AlwaysRebuild

	name resource.Name

	board board.Board

	logger logging.Logger
	cfg    *Config

	cancelCtx  context.Context
	cancelFunc func()

	pulseLength int
	txRepeat    int
}

func newRfTransmitter433MhzRfTransmitter(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (resource.Resource, error) {
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}

	return NewRfTransmitter(ctx, deps, rawConf.ResourceName(), conf, logger)

}

func NewRfTransmitter(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *Config, logger logging.Logger) (resource.Resource, error) {

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	b, err := deps.Lookup(board.Named(conf.Board))
	if err != nil {
		return nil, err
	}

	s := &rfTransmitter433MhzRfTransmitter{
		name:        name,
		logger:      logger,
		cfg:         conf,
		cancelCtx:   cancelCtx,
		cancelFunc:  cancelFunc,
		board:       b.(board.Board),
		pulseLength: 350, // microseconds
		txRepeat:    10,
	}
	return s, nil
}

func (s *rfTransmitter433MhzRfTransmitter) Name() resource.Name {
	return s.name
}

func (s *rfTransmitter433MhzRfTransmitter) NewClientFromConn(ctx context.Context, conn rpc.ClientConn, remoteName string, name resource.Name, logger logging.Logger) (resource.Resource, error) {
	panic("not implemented")
}

func (s *rfTransmitter433MhzRfTransmitter) transmitWaveform(ctx context.Context, highPulses int, lowPulses int, pin board.GPIOPin) (bool, error) {
	err := pin.Set(ctx, true, nil)
	if err != nil {
		return false, err
	}
	time.Sleep(time.Duration(highPulses * s.pulseLength * 1000)) // nanoseconds
	err = pin.Set(ctx, false, nil)
	if err != nil {
		return false, err
	}
	time.Sleep(time.Duration(lowPulses * s.pulseLength * 1000)) // nanoseconds
	return true, nil
}

func (s *rfTransmitter433MhzRfTransmitter) transmitZero(ctx context.Context, pin board.GPIOPin) (bool, error) {
	return s.transmitWaveform(ctx, 1, 3, pin)
}

func (s *rfTransmitter433MhzRfTransmitter) transmitOne(ctx context.Context, pin board.GPIOPin) (bool, error) {
	return s.transmitWaveform(ctx, 3, 1, pin)
}

func (s *rfTransmitter433MhzRfTransmitter) transmitSync(ctx context.Context, pin board.GPIOPin) (bool, error) {
	return s.transmitWaveform(ctx, 1, 31, pin)
}

func (s *rfTransmitter433MhzRfTransmitter) transmit(ctx context.Context, code int64) (bool, error) {
	pin, err := s.board.GPIOPinByName(s.cfg.DataPin)
	if err != nil {
		return false, err
	}

	rawCode := fmt.Sprintf("%024b", code)

	s.logger.CInfof(ctx, "Transmitting code %v", code)
	s.logger.CInfof(ctx, "Transmitting binary %v", rawCode)
	for i := 0; i < s.txRepeat; i++ {
		for _, char := range rawCode {
			if char == '0' {
				_, err := s.transmitZero(ctx, pin)
				if err != nil {
					return false, err
				}
			}
			if char == '1' {
				_, err := s.transmitOne(ctx, pin)
				if err != nil {
					return false, err
				}
			}
		}
	}
	// transmit sync
	_, err = s.transmitSync(ctx, pin)
	if err != nil {
		return false, err
	}
	s.logger.CInfo(ctx, "Transmission successful")
	return true, nil
}

func (s *rfTransmitter433MhzRfTransmitter) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	cmdName, ok := cmd["command"]
	if !ok {
		return map[string]interface{}{}, errors.New("command is required")
	}
	if cmdName == "transmit" {
		code, ok := cmd["code"]
		if !ok {
			return map[string]interface{}{}, errors.New("code is required")
		}
		if codeInt, ok := code.(int64); ok {
			success, err := s.transmit(ctx, codeInt)
			if success {
				return map[string]interface{}{"success": true}, nil
			} else {
				return map[string]interface{}{"error": err}, err
			}
		} else {
			return map[string]interface{}{}, errors.New(fmt.Sprintf("code must be an integer: %v", code))
		}
	} else {
		return map[string]interface{}{}, errors.New("unsupported command")
	}
}

func (s *rfTransmitter433MhzRfTransmitter) Close(context.Context) error {
	// Put close code here
	s.cancelFunc()
	return nil
}
