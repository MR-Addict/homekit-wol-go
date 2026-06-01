package wol

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func BuildMagicPacket(mac net.HardwareAddr) ([]byte, error) {
	if len(mac) != 6 {
		return nil, fmt.Errorf("mac address must be 6 bytes")
	}

	packet := make([]byte, 6+(16*len(mac)))
	for index := 0; index < 6; index++ {
		packet[index] = 0xFF
	}
	for index := 6; index < len(packet); index += len(mac) {
		copy(packet[index:index+len(mac)], mac)
	}

	return packet, nil
}

func Send(ctx context.Context, mac, broadcastIP string, port int) error {
	hardwareAddr, err := net.ParseMAC(strings.TrimSpace(mac))
	if err != nil {
		return fmt.Errorf("parse mac address: %w", err)
	}

	packet, err := BuildMagicPacket(hardwareAddr)
	if err != nil {
		return err
	}

	address := net.JoinHostPort(strings.TrimSpace(broadcastIP), strconv.Itoa(port))
	dialer := net.Dialer{
		Control: enableBroadcastSocket,
	}

	conn, err := dialer.DialContext(ctx, "udp4", address)
	if err != nil {
		return fmt.Errorf("dial broadcast address: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}
	}

	written, err := conn.Write(packet)
	if err != nil {
		return fmt.Errorf("send magic packet: %w", err)
	}
	if written != len(packet) {
		return fmt.Errorf("send magic packet: wrote %d of %d bytes", written, len(packet))
	}

	return nil
}
