package kasa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Version of this library.
var Version = "development"

// The following encrypt and decrupt functions are taken from
// https://github.com/joeshaw/kasa-homekit

const initialKey = byte(0xab)

func encrypt(data []byte) []byte {
	out := make([]byte, len(data))
	key := initialKey
	for i := range data {
		out[i] = data[i] ^ key
		key = out[i]
	}
	return out
}

func decrypt(data []byte) []byte {
	out := make([]byte, len(data))
	key := initialKey
	for i := range data {
		out[i] = data[i] ^ key
		key = data[i]
	}
	return out
}

// APIMessage wraps requests to and responses from Kasa devices. Does not
// currently support emeter details.
type APIMessage struct {
	RemoteAddress *net.UDPAddr           `json:"-"`
	System        map[string]interface{} `json:"system"`
}

// Encode an API message into the wire format expected by Kasa devices. This is
// a JSON payload with no trailing whitespace, "encrypted" using the encrypt
// function.
func (p *APIMessage) Encode() ([]byte, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(p); err != nil {
		return nil, err
	}
	msg := b.Bytes()
	// Strip the trailing newline, kasa devices do not like that because it messes
	// with the "encryption."
	msg = msg[:len(msg)-1]
	return encrypt(msg), nil
}

// GetModule from the APIMessage. A "module" is the command or response to a
// command inside the system object of the Kasa API message. Examples include
// get_sysinfo and set_relate_state.
//
// This is a utility intended for use with mapstructure or similar.
func (p *APIMessage) GetModule(module string) (map[string]interface{}, bool) {
	if p == nil {
		return nil, false
	}
	r, has := p.System[module]
	if !has {
		return nil, false
	}
	mod, ok := r.(map[string]interface{})
	return mod, ok
}

// DecodeAPIMessage from the "encrypted" Kasa wire format.
func DecodeAPIMessage(raw []byte, message *APIMessage) error {
	return json.Unmarshal(decrypt(raw), message)
}

// receive attempts to read APIMessages from a UDP connection.
func receive(ctx context.Context, conn *net.UDPConn) ([]*APIMessage, error) {
	var replies []*APIMessage
	buf := make([]byte, 2048)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return replies, err
		}

		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok {
				if nerr.Timeout() {
					return replies, nil
				}
			}
			return replies, err
		}

		var reply APIMessage
		if err := DecodeAPIMessage(buf[:n], &reply); err != nil {
			continue
		}
		reply.RemoteAddress = raddr
		replies = append(replies, &reply)
	}
}

// Send an APIMessage to a UDP address. The address can be either an individual
// Kasa device's address, or a broadcast address. If expectResponse is true, then
// receive is called and any responses are returned. If expectResponse is false,
// the returned APIMessage slice will always be nil.
func Send(ctx context.Context, message *APIMessage, raddr, laddr *net.UDPAddr, expectResponse bool) ([]*APIMessage, error) {
	msg, err := message.Encode()
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil, err
	}
	if _, err = conn.WriteToUDP(msg, raddr); err != nil {
		return nil, err
	}
	if !expectResponse {
		return nil, nil
	}
	return receive(ctx, conn)
}

// ErrGetSysinfoFailed is returned by a Kasa device when get_sysinfo fails.
var ErrGetSysinfoFailed = errors.New("get_sysinfo failed")

// SystemInformation gives structure to the response to get_sysinfo requests.
type SystemInformation struct {
	RemoteAddress *net.UDPAddr `json:"-"`

	ErrorCode int    `json:"err_code,omitempty" mapstructure:"err_code"`
	Error     string `json:"error_msg,omitempty" mapstructure:"error_msg"`

	ActiveMode      string `json:"active_mode,omitempty" mapstructure:"active_mode"`
	Alias           string `json:"alias,omitempty" mapstructure:"alias"`
	DeviceID        string `json:"deviceId,omitempty" mapstructure:"deviceId"`
	DevName         string `json:"dev_name,omitempty" mapstructure:"dev_name"`
	Feature         string `json:"feature,omitempty" mapstructure:"feature"`
	HardwareID      string `json:"hwId,omitempty" mapstructure:"hwId"`
	HardwareVersion string `json:"hw_ver,omitempty" mapstructure:"hw_ver"`
	IconHash        string `json:"icon_hash,omitempty" mapstructure:"icon_hash"`
	LEDOff          int    `json:"led_off,omitempty" mapstructure:"led_off"`
	MAC             string `json:"mac,omitempty" mapstructure:"mac"`
	MicType         string `json:"mic_type,omitempty" mapstructure:"mic_type"`
	Model           string `json:"model,omitempty" mapstructure:"model"`
	NTCCode         int    `json:"ntc_code,omitempty" mapstructure:"ntc_code"`
	OEMID           string `json:"oemId,omitempty" mapstructure:"oemId"`
	OnTime          int    `json:"on_time,omitempty" mapstructure:"on_time"`
	RelayState      int    `json:"relay_state,omitempty" mapstructure:"relay_state"`
	RSSI            int    `json:"rssi,omitempty" mapstructure:"rssi"`
	SoftwareVersion string `json:"sw_ver,omitempty" mapstructure:"sw_ver"`
	Status          string `json:"status,omitempty" mapstructure:"status"`
	Updating        int    `json:"updating,omitempty" mapstructure:"updating"`

	NextAction *struct {
		Type int `json:"type,omitempty"`
	} `json:"next_action,omitempty"`
}

// Err converts any error details in a get_sysinfo response to a Go error.
func (p SystemInformation) Err() error {
	if code := p.ErrorCode; code != 0 {
		if em := p.Error; em != "" {
			return fmt.Errorf("%w: error code %v: %v", ErrGetSysinfoFailed, code, em)
		}
		return fmt.Errorf("%w: error code %v", ErrGetSysinfoFailed, code)
	}
	return nil
}

// FromAPIMessage populates a SystemInformation from an APIMessage.
func (i *SystemInformation) FromAPIMessage(msg *APIMessage) error {
	mr, ok := msg.GetModule("get_sysinfo")
	if !ok {
		return fmt.Errorf("%w: response did not contain get_sysinfo payload", ErrGetSysinfoFailed)
	}
	if err := mapstructure.Decode(mr, i); err != nil {
		return err
	}
	i.RemoteAddress = msg.RemoteAddress
	return nil
}

// GetSystemInformation sends a get_sysinfo request to the UDP address, and
// returns any responses received before the deadline.
func GetSystemInformation(ctx context.Context, raddr, laddr *net.UDPAddr, allOrNothing bool) ([]*SystemInformation, error) {
	message := &APIMessage{
		System: map[string]interface{}{
			"get_sysinfo": nil,
		},
	}
	replies, err := Send(ctx, message, raddr, laddr, true)
	if err != nil {
		return nil, err
	}
	var r []*SystemInformation
	for _, reply := range replies {
		var si SystemInformation
		if err := si.FromAPIMessage(reply); err != nil {
			if allOrNothing {
				return nil, err
			}
			continue
		}
		r = append(r, &si)
	}
	return r, nil
}

type setRelayStateRequest struct {
	State bool `json:"state" mapstructure:"state"`
}

// SetRelayState on the specified address.
func SetRelayState(ctx context.Context, raddr, laddr *net.UDPAddr, state bool) error {
	message := &APIMessage{
		System: map[string]interface{}{
			"set_relay_state": setRelayStateRequest{
				State: state,
			},
		},
	}
	_, err := Send(ctx, message, raddr, laddr, false)
	return err
}
