package device

import (
	"errors"
	"fmt"
	"strings"

	"github.com/robherley/etherlighter/internal/config"
	"golang.org/x/crypto/ssh"
)

func Connect(cfg *config.Config) (*Client, error) {
	sshCfg, err := cfg.ToSSHConfig()
	if err != nil {
		return nil, err
	}

	client := &Client{
		addr: cfg.DeviceAddr,
		cfg:  sshCfg,
	}

	if err := client.Dial(); err != nil {
		return nil, err
	}

	return client, nil
}

type Client struct {
	addr string
	cfg  *ssh.ClientConfig
	ssh  *ssh.Client
}

func (client *Client) Close() error {
	return client.ssh.Close()
}

func (client *Client) Dial() error {
	var err error

	client.ssh, err = ssh.Dial("tcp", client.addr, client.cfg)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) exec(cmd string) (string, error) {
	session, err := client.ssh.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	raw, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}

	return string(raw), nil
}

type ClientInfo struct {
	Hostname string  `json:"hostname"`
	IP       string  `json:"ip"`
	MAC      string  `json:"mac"`
	Model    string  `json:"model"`
	NTP      string  `json:"ntp"`
	Status   string  `json:"status"`
	Uptime   string  `json:"uptime"`
	Version  string  `json:"version"`
	Layout   [][]int `json:"layout"`
}

func (client *Client) Info() (*ClientInfo, error) {
	output, err := client.exec("mca-cli-op info")
	if err != nil {
		return nil, err
	}

	info := &ClientInfo{}

	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch key {
		case "Hostname":
			info.Hostname = value
		case "IP Address":
			info.IP = value
		case "MAC Address":
			info.MAC = value
		case "Model":
			info.Model = value
		case "NTP":
			info.NTP = value
		case "Status":
			info.Status = value
		case "Uptime":
			info.Uptime = value
		case "Version":
			info.Version = value
		}
	}

	switch info.Model {
	case "USW-Pro-Max-24-PoE":
		info.Layout = [][]int{
			toRange(1, 24, 1),
		}
	// all these others are guesses, because I don't own them
	// this is not exhaustive list of all etherlight-enabled models
	case "USW-Pro-Max-48-PoE":
		info.Layout = [][]int{
			toRange(1, 47, 2),
			toRange(2, 48, 2),
		}
	case "USW-Pro-Max-16-PoE":
		info.Layout = [][]int{
			toRange(1, 16, 1),
		}
	case "USW-Pro-Max-48":
		info.Layout = [][]int{
			toRange(1, 47, 2),
			toRange(2, 48, 2),
		}
	case "USW-Pro-Max-24":
		info.Layout = [][]int{
			toRange(1, 24, 1),
		}
	case "USW-Pro-Max-16":
		info.Layout = [][]int{
			toRange(1, 16, 1),
		}
	}

	return info, nil
}

type SystemConfigOutput struct {
	Etherlight struct {
		Behavior   string `json:"behavior"`
		Brightness string `json:"brightness"`
		Mode       string `json:"mode"`
		// TODO: see if adding modes & color/types is worth
	} `json:"etherlight"`
}

func (client *Client) SystemConfig() (*SystemConfigOutput, error) {
	output, err := client.exec("cat /tmp/system.cfg")
	if err != nil {
		return nil, err
	}

	conf := &SystemConfigOutput{}

	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		if !strings.HasPrefix(parts[0], "switch.etherlight.") {
			continue
		}

		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

		switch key {
		case "switch.etherlight.behavior":
			conf.Etherlight.Behavior = value
		case "switch.etherlight.brightness":
			conf.Etherlight.Brightness = value
		case "switch.etherlight.mode":
			conf.Etherlight.Mode = value
		}
	}

	return conf, nil
}

type Mode string

const (
	ModeColdReset       Mode = "cold_reset"
	ModeWarmReset       Mode = "warm_reset"
	ModeBootDone        Mode = "boot_done"
	ModeSpeed           Mode = "speed"
	ModePOE             Mode = "poe"
	ModeNetwork         Mode = "network"
	ModeDeviceType      Mode = "device_type"
	ModePortLocate      Mode = "port_locate"
	ModePortLocateUnset Mode = "port_locate_unset"
)

func (client *Client) SetMode(mode Mode) error {
	if err := client.setLEDMode(1); err != nil {
		return err
	}

	var args []string

	switch mode {
	case ModeColdReset:
		// 0: "Cold reset" - rainbow back and forth animation
		args = []string{"0"}
	case ModeWarmReset:
		// 1: "Warm reset" - white breathing animation
		args = []string{"1"}
	case ModeBootDone:
		// 2: Boot done - normal operation, uses the 10-modes below
		args = []string{"2"}
	case ModeSpeed:
		// 10: Speed - 0: speed, 1: network, 2: poe, 3: device_type, 4: port_locate, 5: port_locate_unset
		args = []string{"10", "0"}
	case ModeNetwork:
		args = []string{"10", "1"}
	case ModePOE:
		args = []string{"10", "2"}
	case ModeDeviceType:
		args = []string{"10", "3"}
	case ModePortLocate:
		args = []string{"10", "4"}
	case ModePortLocateUnset:
		args = []string{"10", "5"}
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}

	_, err := client.exec(fmt.Sprintf("echo %s > /proc/led/led_config", strings.Join(args, " ")))
	if err != nil {
		return err
	}

	return nil
}

type Color struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

type PortColor struct {
	Index int   `json:"index"`
	Color Color `json:"color"`
}

func (client *Client) SetAllPorts(color Color, brightness uint8) error {
	if err := client.setLEDMode(0); err != nil {
		return err
	}

	if brightness > 100 {
		return errors.New("brightness must be between 0 and 100")
	}

	// cat /proc/led/led_all_port_code
	// * Set all ports' LED color code for r/g/b [00-FF] with power level [0-100]
	// * Ex. "FF 00 FF 100" to set all ports to color code r=FF g=00 b=FF with power level 100
	cmd := fmt.Sprintf("echo '%02X %02X %02X %d' > led_all_port_code", color.R, color.G, color.B, brightness)
	if _, err := client.exec(cmd); err != nil {
		return err
	}

	return nil
}

func (client *Client) SetPortColors(portColors []PortColor) error {
	if err := client.setLEDMode(0); err != nil {
		return err
	}

	cmds := make([]string, 0, len(portColors))
	for _, portColor := range portColors {

		// unfortunately the "bulk" led_code file doesn't work... the color is always red-ish
		// cat /proc/led/led_code
		// * Set port[1-52] LED with color code r[0-ff] g[0-ff] b[0-ff] and power level[1-100]
		// * Ex. "1 ff cc ff 100" to light port 1 with color code #ffccff and power level 100

		// we'll have to set them individually with /proc/led/led_color
		rgb := []string{
			fmt.Sprintf("echo %d r %d > /proc/led/led_color", portColor.Index, int(portColor.Color.R)*100),
			fmt.Sprintf("echo %d g %d > /proc/led/led_color", portColor.Index, int(portColor.Color.G)*100),
			fmt.Sprintf("echo %d b %d > /proc/led/led_color", portColor.Index, int(portColor.Color.B)*100),
		}

		cmds = append(cmds, strings.Join(rgb, " && "))
	}

	// this can get long, maybe chunk them eventually?
	joined := strings.Join(cmds, " && ")
	if _, err := client.exec(joined); err != nil {
		return err
	}

	return nil
}

func (client *Client) setLEDMode(mode uint8) error {
	// 0: resets all ports and enables writing?
	// 1: resets all ports and enables writing to _connected_ ports only?
	if _, err := client.exec(fmt.Sprintf("echo %d > /proc/led/led_mode", mode)); err != nil {
		return err
	}

	return nil
}

// toRange generates a range of integers, inclusive, with a skip value
func toRange(start, end, skip int) []int {
	var result []int
	for i := start; i <= end; i += skip {
		result = append(result, i)
	}
	return result
}
